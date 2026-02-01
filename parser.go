package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func extractZip(src, dest string) (string, error) {
	Logger.Debug("Extracting ZIP file", "source", src, "destination", dest)

	r, err := zip.OpenReader(src)
	if err != nil {
		Logger.Error("Failed to open ZIP file", "error", err, "source", src)
		return "", err
	}
	defer r.Close()

	var chatFile string
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", f.Name)
		}

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

	Logger.Debug("ZIP extraction completed", "chat_file", chatFile)
	return chatFile, nil
}

var imageRegex = regexp.MustCompile(`(?i)[\w\-.]+\.(jpg|jpeg|png|gif|webp)`)

func parseChat(chatFile, baseDir, staticRoot string) ([]Message, error) {
	Logger.Debug("Parsing chat file", "file", chatFile)

	file, err := os.Open(chatFile)
	if err != nil {
		Logger.Error("Failed to open chat file", "error", err, "file", chatFile)
		return nil, err
	}
	defer file.Close()

	var messages []Message
	var user1 string

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024) // 1MB lines
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

		if text == "" || text == "null" {
			continue
		}

		if user1 == "" {
			user1 = sender
		}

		var imagePath string
		match := imageRegex.FindString(text)
		if match != "" {
			cleanName := strings.TrimSpace(strings.Map(func(r rune) rune {
				if r == '\u200E' || r == '\u200F' || r == '\uFEFF' {
					return -1 // remove invisible marks
				}
				return r
			}, match))
			candidate := filepath.Join(baseDir, cleanName)
			if _, err := os.Stat(candidate); err == nil {
				rel, err := filepath.Rel(staticRoot, candidate)
				if err == nil {
					imagePath = rel
				}
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

	Logger.Info("Chat parsing completed",
		"file", chatFile,
		"messages_count", len(messages),
		"images_count", func() int {
			count := 0
			for _, msg := range messages {
				if msg.ImagePath != "" {
					count++
				}
			}
			return count
		}())

	return messages, scanner.Err()
}
