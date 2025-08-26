// This tool is intended for authorized security testing only.
// Users must ensure they have permission to test target systems.

This is a re-write of the ruby aws_bucketfinder in go, and adding in concurrency use -w to specify workers,
default is 10. I also added a -k flag like in gcpBukcetfinder that can iterate off a single word using common variants. 
