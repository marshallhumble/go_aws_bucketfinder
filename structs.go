package main

import (
	"encoding/xml"
	"log"
	"time"
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
	KeyCount              int    `xml:"KeyCount"`
	IsTruncated           bool   `xml:"IsTruncated"`
	NextContinuationToken string `xml:"NextContinuationToken"`
}

type S3Error struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Endpoint string   `xml:"Endpoint"`
}

type Config struct {
	download   bool
	logFile    string
	region     string
	verbose    bool
	wordlist   string
	keyword    string
	workers    int
	logger     *log.Logger
	rateLimit  time.Duration
	maxKeys    int
	maxObjects int
}
