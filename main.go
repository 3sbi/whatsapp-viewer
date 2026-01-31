package main

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	e := echo.New()
	e.Use(middleware.Logger())

	baseTmp := "tmp"
	if err := os.MkdirAll(baseTmp, os.ModePerm); err != nil {
		log.Fatal(err)
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
		log.Fatal("Error loading upload template:", err)
	}

	e.GET("/", func(c echo.Context) error {
		return uploadTemplate.Execute(c.Response(), nil)
	})

	e.POST("/upload", func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}

		sessionID := getSessionID(c)

		// Create temporary directory for processing only
		tmpDir, err := os.MkdirTemp(baseTmp, "chat")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir) // Clean up immediately after processing

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

		// Store session data
		session := &ChatSession{
			Messages:  messages,
			ImageData: make(map[string][]byte),
			CreatedAt: time.Now(),
		}

		// Load image data into memory
		for _, msg := range messages {
			if msg.ImagePath != "" {
				imagePath := filepath.Join(tmpDir, filepath.Base(msg.ImagePath))
				if data, err := os.ReadFile(imagePath); err == nil {
					session.ImageData[msg.ImagePath] = data
				}
			}
		}

		// Compute session size once
		var size int64
		for _, img := range session.ImageData {
			size += int64(len(img))
		}
		session.SizeBytes = size

		sessionStore.Set(sessionID, session)

		// Check memory usage after storing new session
		sessionStore.Cleanup()

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

		// Detect actual content type from file
		contentType := http.DetectContentType(imageData[:512])
		return c.Blob(http.StatusOK, contentType, imageData)
	})

	e.Logger.Fatal(e.Start(":5556"))
}
