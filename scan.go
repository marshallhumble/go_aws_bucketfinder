package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func processBucketsWithWorkers(config *Config, host string, bucketNames []string) {
	jobs := make(chan string, len(bucketNames))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			for bucketName := range jobs {
				if config.verbose {
					fmt.Printf("[Worker %d] Checking bucket: %s\n", workerId, bucketName)
				}

				// Rate limiting
				time.Sleep(config.rateLimit)

				data, err := getPage(host, bucketName)
				if err != nil {
					if config.verbose {
						fmt.Printf("[Worker %d] Error requesting page for %s: %v\n", workerId, bucketName, err)
					}
					if config.logger != nil {
						config.logger.Printf("[Worker %d] Error requesting page for %s: %v", workerId, bucketName, err)
					}
					continue
				}

				if data != "" {
					parseResults(config, data, bucketName, host, 0, workerId)
				}
			}
		}(i)
	}

	// Send jobs
	for _, bucketName := range bucketNames {
		jobs <- bucketName
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
}

func getPage(host, page string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	pageURL := fmt.Sprintf("%s/%s", host, page)
	resp, err := client.Get(pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
