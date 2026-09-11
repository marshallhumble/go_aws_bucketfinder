package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func parseResults(config *Config, data, bucketName, host string, depth, workerId int) {
	tabs := strings.Repeat("\t", depth)
	workerPrefix := ""
	if config.verbose {
		workerPrefix = fmt.Sprintf("[Worker %d] ", workerId)
	}

	// Try to parse as ListBucketResult first
	var listResult ListBucketResult
	if err := xml.Unmarshal([]byte(data), &listResult); err == nil && listResult.Name != "" {
		msg := fmt.Sprintf("%s%sBucket Found: %s ( %s/%s )", workerPrefix, tabs, bucketName, host, bucketName)
		fmt.Println(msg)
		if config.logger != nil {
			config.logger.Println(msg)
		}

		for _, content := range listResult.Contents {
			processFile(config, content.Key, bucketName, host, depth, workerId)
		}
		return
	}

	// Try to parse as error
	var s3Error S3Error
	if err := xml.Unmarshal([]byte(data), &s3Error); err == nil && s3Error.Code != "" {
		handleS3Error(config, s3Error, bucketName, host, depth, workerId)
		return
	}

	if config.verbose {
		msg := fmt.Sprintf("%s%s No valid data returned", workerPrefix, tabs)
		fmt.Println(msg)
		if config.logger != nil {
			config.logger.Println(msg)
		}
	}
}

func processFile(config *Config, key, bucketName, host string, depth, workerId int) {
	tabs := strings.Repeat("\t", depth+1)
	workerPrefix := ""
	if config.verbose {
		workerPrefix = fmt.Sprintf("[Worker %d] ", workerId)
	}

	// Build URL
	var fileURL string
	if strings.HasPrefix(host, "http") {
		if strings.Contains(host, bucketName) {
			fileURL = fmt.Sprintf("%s/%s", host, url.QueryEscape(key))
		} else {
			fileURL = fmt.Sprintf("%s/%s/%s", host, bucketName, url.QueryEscape(key))
		}
	} else {
		fileURL = fmt.Sprintf("http://%s/%s/%s", host, bucketName, url.QueryEscape(key))
	}

	// Skip directories (keys ending with /)
	if strings.HasSuffix(key, "/") {
		return
	}

	readable := false
	downloaded := false

	if config.download && key != "" {
		downloaded, readable = downloadFile(fileURL, bucketName, key, depth)
	} else {
		readable = checkFileReadable(fileURL)
	}

	var msg string
	if readable {
		if downloaded {
			msg = fmt.Sprintf("%s%s<Downloaded> %s", workerPrefix, tabs, fileURL)
		} else {
			msg = fmt.Sprintf("%s%s<Public> %s", workerPrefix, tabs, fileURL)
		}
	} else {
		msg = fmt.Sprintf("%s%s<Private> %s", workerPrefix, tabs, fileURL)
	}

	fmt.Println(msg)
	if config.logger != nil {
		config.logger.Println(msg)
	}
}

func checkFileReadable(fileURL string) bool {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Head(fileURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func handleS3Error(config *Config, s3Error S3Error, bucketName, host string, depth, workerId int) {
	tabs := strings.Repeat("\t", depth)
	workerPrefix := ""
	if config.verbose {
		workerPrefix = fmt.Sprintf("[Worker %d] ", workerId)
	}

	var msg string

	switch s3Error.Code {
	case "NoSuchKey":
		msg = fmt.Sprintf("%s%sThe specified key does not exist: %s", workerPrefix, tabs, bucketName)
	case "AccessDenied":
		msg = fmt.Sprintf("%s%sBucket found but access denied: %s", workerPrefix, tabs, bucketName)
	case "NoSuchBucket":
		if config.verbose {
			msg = fmt.Sprintf("%s%sBucket does not exist: %s", workerPrefix, tabs, bucketName)
			fmt.Println(msg)
		}
		// Don't log non-existent buckets to keep output clean
		return
	case "PermanentRedirect":
		if s3Error.Endpoint != "" {
			msg = fmt.Sprintf("%s%sBucket %s redirects to: %s", workerPrefix, tabs, bucketName, s3Error.Endpoint)
			fmt.Println(msg)
			if config.logger != nil {
				config.logger.Println(msg)
			}

			// Follow redirect
			fmt.Printf("%s%sFollowing redirect...\n", workerPrefix, tabs)
			data, err := getPage("https://"+s3Error.Endpoint, "")
			if err != nil {
				fmt.Printf("%s%sError following redirect: %v\n", workerPrefix, tabs, err)
				return
			}
			if data != "" {
				fmt.Printf("%s%sChecking redirected bucket:\n", workerPrefix, tabs)
				parseResults(config, data, bucketName, s3Error.Endpoint, depth+1, workerId)
			}
			return
		} else {
			msg = fmt.Sprintf("%s%sRedirect found but can't find where to: %s", workerPrefix, tabs, bucketName)
		}
	default:
		msg = fmt.Sprintf("%s%sUnknown error for %s: %s - %s", workerPrefix, tabs, bucketName, s3Error.Code, s3Error.Message)
	}

	fmt.Println(msg)
	if config.logger != nil {
		config.logger.Println(msg)
	}
}
