package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

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
