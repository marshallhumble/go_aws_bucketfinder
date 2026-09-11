package main

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
