# AWS S3 Bucket Finder

A high-performance Go rewrite of the classic Ruby [aws_bucketfinder](https://github.com/digininja/bucket_finder) with enhanced features for modern cloud security testing.

## Disclaimer

This tool is for authorized testing only. Users are responsible for:
- Obtaining proper authorization before testing
- Complying with applicable laws and regulations
- Using the tool ethically and responsibly

The authors assume no liability for misuse of this tool.

## Features

- **Concurrent Processing**: Multi-threaded bucket enumeration with configurable workers (`-w` flag, default: 10)
- **Smart Permutations**: Keyword-based bucket name generation (`-k` flag) inspired by [GCPBucketBrute](https://github.com/RhinoSecurityLabs/GCPBucketBrute)
- **Multi-Region Support**: Test buckets across different AWS regions
- **File Download**: Automatically download publicly accessible files
- **Comma-Separated Keywords**: Generate permutations from multiple keywords
- **Real-time Logging**: Optional file logging with timestamps

```
--help, -h:        Show help
--download, -d:    Download any public files found
--log-file, -l:    Filename to log output to
--region, -r:      AWS region (us, ie, nc, si, to)
--keyword, -k:     Generate bucket names from keyword permutations
--workers, -w:     Number of concurrent workers (default: 10)
-v:               Verbose output
```

## Examples

### Use wordlist file with 5 workers
./bucket_finder -w 5 *wordlist.txt*

### Generate permutations from keyword
./bucket_finder -k "company" -w 10

### Multiple keywords with file download
./bucket_finder -k "acme,corp,example.com" -d -w 15

### Specific region with logging
./bucket_finder -k "company" -r "ie" -l results.log -w 20



## Installation

```bash
git clone https://github.com/marshallhumble/go_aws_bucketfinder
cd go_aws_bucketfinder
go build -o bucket_finder main.go
```
