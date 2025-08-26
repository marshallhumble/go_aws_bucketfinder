package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	version = "2.1"
	author  = "Converted to Go from Robin Wood's original Ruby script"
)

// S3 XML response structures
type ListBucketResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Name     string   `xml:"Name"`
	Contents []struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		ETag         string `xml:"ETag"`
		Size         int64  `xml:"Size"`
	} `xml:"Contents"`
}

type S3Error struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Endpoint string   `xml:"Endpoint"`
}

type Config struct {
	download  bool
	logFile   string
	region    string
	verbose   bool
	wordlist  string
	keyword   string
	workers   int
	logger    *log.Logger
	rateLimit time.Duration
}

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

func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.download, "download", false, "Download any public files found")
	flag.BoolVar(&config.download, "d", false, "Download any public files found (shorthand)")
	flag.StringVar(&config.logFile, "log-file", "", "Filename to log output to")
	flag.StringVar(&config.logFile, "l", "", "Filename to log output to (shorthand)")
	flag.StringVar(&config.region, "region", "us", "The region to use (us, ie, nc, si, to)")
	flag.StringVar(&config.region, "r", "us", "The region to use (shorthand)")
	flag.StringVar(&config.keyword, "keyword", "", "Generate bucket names from keyword permutations")
	flag.StringVar(&config.keyword, "k", "", "Generate bucket names from keyword permutations (shorthand)")
	flag.IntVar(&config.workers, "workers", 10, "Number of concurrent workers")
	flag.IntVar(&config.workers, "w", 10, "Number of concurrent workers (shorthand)")
	flag.BoolVar(&config.verbose, "v", false, "Verbose output")

	help := flag.Bool("help", false, "Show help")
	helpShort := flag.Bool("h", false, "Show help (shorthand)")

	flag.Parse()

	if *help || *helpShort {
		usage()
		os.Exit(0)
	}

	// Set rate limit based on number of workers to avoid overwhelming S3
	config.rateLimit = time.Duration(1000/config.workers) * time.Millisecond

	if flag.NArg() == 1 && config.keyword == "" {
		config.wordlist = flag.Arg(0)
	}

	return config
}

func usage() {
	fmt.Printf(`bucket_finder %s - %s

Usage: bucket_finder [OPTIONS] [wordlist]
	--help, -h:        Show help
	--download, -d:    Download the files
	--log-file, -l:    Filename to log output to
	--region, -r:      The region to use, options are:
	                   us - US Standard
	                   ie - Ireland  
	                   nc - Northern California
	                   si - Singapore
	                   to - Tokyo
	--keyword, -k:     Generate bucket names from keyword permutations (supports comma or space-separated)
	                   Examples: -k "company" or -k "acme,corp" or -k "findhelp auntbertha"
	--workers, -w:     Number of concurrent workers (default: 10)
	-v:               Verbose output

	wordlist: The wordlist file to use (optional if using -k/--keyword)

Examples:
	# Use wordlist file
	bucket_finder -w 5 -d wordlist.txt
	
	# Use keyword permutations
	bucket_finder -k "company" -w 10 -l output.log
	
	# Use multiple keywords
	bucket_finder -k "acme,corp,example.com" -w 10 -l output.log
	
	# Use keyword with domain
	bucket_finder -k "google.com,gcp,cloud" -w 15

`, version, author)
}

func getHostForRegion(region string) string {
	switch region {
	case "ie":
		return "https://s3-eu-west-1.amazonaws.com"
	case "nc":
		return "https://s3-us-west-1.amazonaws.com"
	case "us":
		return "https://s3.amazonaws.com"
	case "si":
		return "https://s3-ap-southeast-1.amazonaws.com"
	case "to":
		return "https://s3-ap-northeast-1.amazonaws.com"
	default:
		return ""
	}
}

func loadWordlist(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var names []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			names = append(names, name)
		}
	}

	return names, scanner.Err()
}

func parseKeywords(keywordString string) []string {
	var keywords []string

	// First try comma separation
	if strings.Contains(keywordString, ",") {
		keywords = strings.Split(keywordString, ",")
	} else {
		// If no commas, try space separation
		keywords = strings.Fields(keywordString)
	}

	var cleaned []string
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			cleaned = append(cleaned, keyword)
		}
	}

	return cleaned
}

func generateAllPermutations(keywords []string) []string {
	allPermutations := make(map[string]bool)

	// Debug: print what we're processing
	if len(keywords) == 1 {
		fmt.Printf("Processing single keyword: '%s'\n", keywords[0])
	} else {
		fmt.Printf("Processing %d keywords: %v\n", len(keywords), keywords)
	}

	// Generate permutations for each individual keyword
	for i, keyword := range keywords {
		keywordPerms := generateSingleWordPermutations(keyword)
		fmt.Printf("Keyword '%s' generated %d permutations\n", keyword, len(keywordPerms))

		for _, perm := range keywordPerms {
			allPermutations[perm] = true
		}

		fmt.Printf("Total unique permutations after keyword %d: %d\n", i+1, len(allPermutations))
	}

	// Generate cross-keyword combinations for multi-keyword inputs
	if len(keywords) > 1 {
		beforeCross := len(allPermutations)
		generateCrossKeywordPermutations(allPermutations, keywords)
		fmt.Printf("Cross-keyword combinations added: %d (total: %d)\n",
			len(allPermutations)-beforeCross, len(allPermutations))
	}

	// Convert to slice
	var result []string
	for name := range allPermutations {
		result = append(result, name)
	}

	return result
}

// New dedicated function to process a single word through all permutation patterns
func generateSingleWordPermutations(word string) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	permutations := make(map[string]bool)

	// Add the base word
	addPermutation(permutations, word)

	// Extract base name from word (for domains and complex inputs)
	baseName := extractBaseName(word)
	if baseName != word {
		addPermutation(permutations, baseName)
	}

	// Generate core permutations for the main word
	generateCorePermutations(permutations, word)

	// If it's a domain, generate domain-specific permutations
	if strings.Contains(word, ".") {
		generateDomainPermutations(permutations, word)
	}

	// Generate year-based permutations (limited set)
	generateYearPermutations(permutations, word)

	// Convert map to slice and filter
	var result []string
	for name := range permutations {
		if isValidBucketName(name) {
			result = append(result, name)
		}
	}

	return result
}

func generateCrossKeywordPermutations(perms map[string]bool, keywords []string) {
	// Generate only the most valuable combinations between keywords
	for i, keyword1 := range keywords {
		base1 := extractBaseName(keyword1)

		for j, keyword2 := range keywords {
			if i >= j { // Avoid duplicates and self-combinations
				continue
			}
			base2 := extractBaseName(keyword2)

			// Only create the most likely cross-combinations
			addPermutation(perms, base1+"-"+base2)
			addPermutation(perms, base2+"-"+base1)

			// Add a few environment-specific cross-combinations
			for _, env := range []string{"prod", "staging", "backup"} {
				addPermutation(perms, base1+"-"+base2+"-"+env)
				addPermutation(perms, base2+"-"+base1+"-"+env)
			}
		}
	}
}

// Legacy function kept for compatibility - now just calls the dedicated single word function
func generatePermutations(keyword string) []string {
	return generateSingleWordPermutations(keyword)
}

func extractBaseName(keyword string) string {
	// Handle domains: "example.com" -> "example"
	if strings.Contains(keyword, ".") {
		parts := strings.Split(keyword, ".")
		if len(parts) > 0 && len(parts[0]) > 2 {
			return parts[0]
		}
	}

	// Handle hyphens and underscores: "acme-corp" -> "acme", "acmecorp"
	for _, sep := range []string{"-", "_", " "} {
		if strings.Contains(keyword, sep) {
			parts := strings.Split(keyword, sep)
			if len(parts) > 0 && len(parts[0]) > 2 {
				return parts[0]
			}
		}
	}

	return keyword
}

func generateCorePermutations(perms map[string]bool, keyword string) {
	// Only generate high-value combinations, not full cartesian product

	// Base keyword with each suffix (most important patterns)
	prioritySuffixes := []string{"", "-prod", "-staging", "-dev", "-backup", "-data", "-api", "-web", "-test", "-logs"}
	for _, suffix := range prioritySuffixes {
		addPermutation(perms, keyword+suffix)
	}

	// Base keyword with each prefix (most important patterns)
	priorityPrefixes := []string{"backup-", "prod-", "staging-", "dev-", "api-", "web-", "test-", "s3-"}
	for _, prefix := range priorityPrefixes {
		addPermutation(perms, prefix+keyword)
	}

	// No-hyphen versions for high-probability patterns
	noHyphenPatterns := []string{
		keyword + "backup", // findhelp + backup = findhelpbackup
		keyword + "prod",
		keyword + "staging",
		keyword + "dev",
		keyword + "data",
		keyword + "test",
		"backup" + keyword, // backup + findhelp = backupfindhelp
		"prod" + keyword,
		"test" + keyword,
		"s3" + keyword,
	}

	for _, pattern := range noHyphenPatterns {
		addPermutation(perms, pattern)
	}

	// A few combined patterns (very selective)
	combinedPatterns := []string{
		"backup-" + keyword + "-prod",
		"prod-" + keyword + "-backup",
		keyword + "-prod-backup",
		keyword + "-staging-backup",
	}

	for _, pattern := range combinedPatterns {
		addPermutation(perms, pattern)
	}

	// Add numbered variations (limited)
	for i := 1; i <= 2; i++ {
		addPermutation(perms, keyword+strconv.Itoa(i))
		addPermutation(perms, keyword+"-"+strconv.Itoa(i))
	}
}

func generateDomainPermutations(perms map[string]bool, keyword string) {
	parts := strings.Split(keyword, ".")
	if len(parts) < 2 {
		return
	}

	domainName := parts[0]

	// Only most common domain variations to avoid explosion
	priorityVariations := []string{"dev", "staging", "prod", "api", "www", "backup"}
	for _, variation := range priorityVariations {
		addPermutation(perms, domainName+"-"+variation)
		addPermutation(perms, variation+"-"+domainName)
	}
}

func generateYearPermutations(perms map[string]bool, keyword string) {
	currentYear := time.Now().Year()

	// Only add current year and previous 2 years
	for year := currentYear - 2; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		addPermutation(perms, keyword+yearStr)
		addPermutation(perms, keyword+"-"+yearStr)
		// Skip year prefix to reduce noise
	}
}

func addPermutation(perms map[string]bool, name string) {
	name = strings.ToLower(name)
	// Remove invalid characters and validate length
	if len(name) >= 3 && len(name) <= 63 && !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-") {
		perms[name] = true
	}
}

func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Basic S3 bucket name validation
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}

	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}

	// Check for valid characters (simplified - just alphanumeric and hyphens)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return false
		}
	}

	return true
}

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

	url := fmt.Sprintf("%s/%s", host, page)
	resp, err := client.Get(url)
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
