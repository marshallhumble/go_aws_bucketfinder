package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func downloadFile(fileURL, bucketName, key string, depth int) (bool, bool) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return false, false
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fileURL)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, false
	}

	// Create directory structure
	fsDir := filepath.Dir(parsedURL.Path)
	if fsDir == "/" {
		fsDir = ""
	} else if fsDir != "" && fsDir[0] == '/' {
		fsDir = fsDir[1:] // Remove leading slash
	}

	if depth > 0 {
		fsDir = filepath.Join(bucketName, fsDir)
	}

	if fsDir != "" {
		if err := os.MkdirAll(fsDir, 0755); err != nil {
			return false, true // Readable but couldn't create dir
		}
	}

	// Download file
	fileName := filepath.Join(fsDir, filepath.Base(key))
	file, err := os.Create(fileName)
	if err != nil {
		return false, true // Readable but couldn't create file
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(fileName) // Clean up partial file
		return false, true  // Readable but couldn't write
	}

	return true, true
}
