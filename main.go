package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

type Message struct {
	Timestamp string
	Sender    string
	Text      string
	ImagePath string
	IsUser1   bool
}

type ChatSession struct {
	Messages  []Message
	ImageData map[string][]byte
	CreatedAt time.Time
	SizeBytes int64
}

var sessionStore = NewSessionStore()

func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getSessionID(c echo.Context) string {
	cookie, err := c.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		sessionID := generateSessionID()
		c.SetCookie(&http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   3600, // 1 hour
			HttpOnly: true,
		})
		return sessionID
	}
	return cookie.Value
}

func main() {
	InitLogger()
	e := echo.New()
	e.Use(StructuredLogger())

	e.Logger.Info("Starting WhatsApp Viewer application")

	baseTmp := "tmp"
	if err := os.MkdirAll(baseTmp, os.ModePerm); err != nil {
		e.Logger.Error("Failed to create temporary directory", "error", err, "path", baseTmp)
		os.Exit(1)
	}

	e.Static("/assets", "assets")

	// Session cleanup
	go func() {
		for {
			time.Sleep(15 * time.Minute)
			// Remove old sessions (older than 1 hour)
			sessionStore.RemoveOldSessions()
		}
	}()

	uploadTemplate, err := template.ParseFiles("templates/upload.html")
	if err != nil {
		e.Logger.Error("Failed to load upload template", "error", err, "template", "upload.html")
		os.Exit(1)
	}

	e.GET("/", func(c echo.Context) error {
		return uploadTemplate.Execute(c.Response(), nil)
	})

	e.POST("/upload", func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			e.Logger.Warn("Upload request missing file", "error", err)
			return err
		}

		sessionID := getSessionID(c)
		e.Logger.Info("Processing file upload",
			"session_id", sessionID,
			"filename", file.Filename,
			"size", file.Size)

		// create temporary directory for processing files before deleting them
		tmpDir, err := os.MkdirTemp(baseTmp, "chat")
		if err != nil {
			return err
		}

		// Clean up IMMEDIATELY after processing and loading it into RAM
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, file.Filename)
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(zipPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

		chatFile, err := extractZip(zipPath, tmpDir)
		if err != nil {
			return err
		}

		messages, err := parseChat(chatFile, tmpDir, tmpDir)
		if err != nil {
			return err
		}

		// store session data
		session := &ChatSession{
			Messages:  messages,
			ImageData: make(map[string][]byte),
			CreatedAt: time.Now(),
		}

		// loading images into memory to not store them on the disk for too long
		for _, msg := range messages {
			if msg.ImagePath != "" {
				imagePath := filepath.Join(tmpDir, filepath.Base(msg.ImagePath))
				if data, err := os.ReadFile(imagePath); err == nil {
					session.ImageData[msg.ImagePath] = data
				}
			}
		}

		// compute session size once to check if there is enough memory later
		var size int64
		for _, img := range session.ImageData {
			size += int64(len(img))
		}
		session.SizeBytes = size

		sessionStore.Set(sessionID, session)

		// Check memory usage after storing new session
		sessionStore.Cleanup()

		e.Logger.Info("File upload processed successfully",
			"session_id", sessionID,
			"messages_processed", len(messages),
			"session_size_mb", float64(session.SizeBytes)/1024/1024)

		chatTemplate, err := template.ParseFiles("templates/chat.html")
		if err != nil {
			return err
		}

		return chatTemplate.Execute(c.Response(), messages)
	})

	// Serve images from session data
	e.GET("/static/:imagePath", func(c echo.Context) error {
		sessionID := getSessionID(c)
		imagePath := c.Param("imagePath")

		session, exists := sessionStore.Get(sessionID)

		if !exists {
			return c.NoContent(http.StatusNotFound)
		}

		imageData, exists := session.ImageData[imagePath]
		if !exists {
			return c.NoContent(http.StatusNotFound)
		}

		contentType := http.DetectContentType(imageData[:512])
		return c.Blob(http.StatusOK, contentType, imageData)
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		port := ":80"
		e.Logger.Info("Starting HTTP server", "port", port)
		if err := e.Start(port); err != nil && err != http.ErrServerClosed {
			e.Logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Gracefully shut down the server with a timeout of 10 seconds.
	<-ctx.Done()
	e.Logger.Info("Shutdown signal received, gracefully shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Error("Failed to shutdown server gracefully", "error", err)
		os.Exit(1)
	}

	e.Logger.Info("Server shutdown completed")
}
