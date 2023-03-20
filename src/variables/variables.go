package variables

import "runtime"

// App name and version
var AppName string
var AppVersion string

// AWS Variables
var AwsRegion string

// CMD
var AwsCmd string

func init() {
	// App name and version
	AppName = "AWS IAM access key rotator"
	AppVersion = "1.0.3"

	// AWS Variables
	AwsRegion = "us-east-1" // IAM is a global service with its endpoint being located in us-east-1

	// Set aws-command based on the OS
	AwsCmd = setAwsCmd()
}

// Set aws-command based on the OS
func setAwsCmd() string {
	if runtime.GOOS == "windows" {
		return "aws.exe"
	} else {
		return "aws"
	}
}
