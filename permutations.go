package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
		keyword + "backup", // copmany + backup = companybackup
		keyword + "prod",
		keyword + "staging",
		keyword + "dev",
		keyword + "data",
		keyword + "test",
		"backup" + keyword, // backup + company = backupcompany
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
