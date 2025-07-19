package main

import (
	"archive/zip"
	"bufio"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

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

func main() {
	e := echo.New()
	e.Use(middleware.Logger())

	baseTmp := "tmp"
	if err := os.MkdirAll(baseTmp, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	e.Static("/static", baseTmp)

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

		tmpDir, err := os.MkdirTemp(baseTmp, "chat")
		if err != nil {
			return err
		}

		defer os.RemoveAll(tmpDir) // clean after serving

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

		messages, err := parseChat(chatFile, tmpDir)
		if err != nil {
			return err
		}

		chatTemplate, err := template.ParseFiles("templates/chat.html")
		if err != nil {
			return err
		}

		return chatTemplate.Execute(c.Response(), messages)
	})

	e.Logger.Fatal(e.Start(":3000"))
}

func extractZip(src, dest string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var chatFile string
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		if _, err = io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return "", err
		}
		outFile.Close()
		rc.Close()

		if strings.HasSuffix(strings.ToLower(f.Name), ".txt") {
			chatFile = fpath
		}
	}

	return chatFile, nil
}

func parseChat(chatFile, baseDir string) ([]Message, error) {
	file, err := os.Open(chatFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []Message
	var user1 string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) < 2 {
			continue
		}
		timestamp := parts[0]
		msgParts := strings.SplitN(parts[1], ": ", 2)
		if len(msgParts) < 2 {
			continue
		}
		sender := msgParts[0]
		text := msgParts[1]

		if user1 == "" {
			user1 = sender
		}

		var imagePath string
		words := strings.Fields(text)
		for _, w := range words {
			lw := strings.ToLower(w)
			if strings.HasPrefix(lw, "img-") && (strings.HasSuffix(lw, ".jpg") || strings.HasSuffix(lw, ".jpeg") || strings.HasSuffix(lw, ".png")) {
				candidate := filepath.Join(baseDir, w)
				if _, err := os.Stat(candidate); err == nil {
					rel, _ := filepath.Rel("tmp", candidate)
					imagePath = rel
				}
				break
			}
		}

		messages = append(messages, Message{
			Timestamp: timestamp,
			Sender:    sender,
			Text:      text,
			ImagePath: imagePath,
			IsUser1:   sender == user1,
		})
	}

	return messages, scanner.Err()
}
