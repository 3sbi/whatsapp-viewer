package main

import (
	"archive/zip"
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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

func parseChat(chatFile, baseDir, staticRoot string) ([]Message, error) {
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
			if strings.HasSuffix(lw, ".jpg") || strings.HasSuffix(lw, ".jpeg") || strings.HasSuffix(lw, ".png") {
				cleanName := strings.TrimSpace(strings.Map(func(r rune) rune {
					if r == '\u200E' || r == '\u200F' || r == '\uFEFF' {
						return -1 // remove invisible marks
					}
					return r
				}, w))
				candidate := filepath.Join(baseDir, cleanName)
				log.Default().Print(os.Stat(candidate))
				if _, err := os.Stat(candidate); err == nil {
					rel, err := filepath.Rel(staticRoot, candidate)
					if err == nil {
						imagePath = rel
					}
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
