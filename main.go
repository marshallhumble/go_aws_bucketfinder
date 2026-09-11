package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	version = "2.2"
	author  = "Converted to Go from Robin Wood's original Ruby script"
)

func main() {
	config := parseFlags()

	if config.wordlist == "" && config.keyword == "" {
		fmt.Println("Missing wordlist or keyword (try --help)")
		os.Exit(1)
	}

	if config.wordlist != "" && config.keyword != "" {
		fmt.Println("Cannot specify both wordlist and keyword, choose one (try --help)")
		os.Exit(1)
	}

	// Setup logging
	if config.logFile != "" {
		logFile, err := os.Create(config.logFile)
		if err != nil {
			fmt.Printf("Could not open the logging file: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
		config.logger = log.New(logFile, "", log.LstdFlags)
	}

	// Get host based on region
	host := getHostForRegion(config.region)
	if host == "" {
		fmt.Println("Unknown region specified")
		usage()
		os.Exit(1)
	}

	var bucketNames []string

	if config.keyword != "" {
		// Generate permutations from keywords (support comma-separated)
		keywords := parseKeywords(config.keyword)
		bucketNames = generateAllPermutations(keywords)
		fmt.Printf("Generated %d bucket name permutations from %d keyword(s): %s\n",
			len(bucketNames), len(keywords), strings.Join(keywords, ", "))
	} else {
		// Load from wordlist file
		if _, err := os.Stat(config.wordlist); os.IsNotExist(err) {
			fmt.Println("Wordlist file doesn't exist")
			usage()
			os.Exit(1)
		}

		var err error
		bucketNames, err = loadWordlist(config.wordlist)
		if err != nil {
			fmt.Printf("Error loading wordlist: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded %d bucket names from wordlist\n", len(bucketNames))
	}

	// Process bucket names with concurrency
	processBucketsWithWorkers(config, host, bucketNames)
}
