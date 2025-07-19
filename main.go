package main

import (
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"

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

		messages, err := parseChat(chatFile, tmpDir, baseTmp)
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
