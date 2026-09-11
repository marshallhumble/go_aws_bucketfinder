package main

import (
	"bufio"
	"os"
	"strings"
)

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
