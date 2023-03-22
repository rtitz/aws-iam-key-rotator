# AWS IAM Key Rotator

This is an AWS IAM Key Rotator created in Go.
It can rotate AWS IAM Keys of one or multiple AWS CLI profiles.
Key rotation can be done in sequence or in a parallel mode, which is much faster if you have many AWS CLI profiles.

Regular key rotation is a best practice for security reasons.


## How to use (pre-compiled binary)
 * In the directory 'bin/' you will find pre-compiled binaries to use. (No need to build it on your own.)
 * Ensure that the AWS CLI is installed and configured! (https://docs.aws.amazon.com/cli/)
 * Help for the parameters:
```
aws-iam-key-rotator -help
```
 * Standard without parallel mode for the 'default' AWS CLI profile
```
aws-iam-key-rotator -profile default
```
 * Multiple AWS CLI profiles without parallel mode:
```
aws-iam-key-rotator -profile default,dev,prod
```
 * Multiple AWS CLI profiles with parallel mode:
```
aws-iam-key-rotator -profile default,dev,prod -parallel
```


## How to build the executable binary (if you don't like the pre-compiled in the 'bin' directory)
 * [Install Go](https://go.dev/doc/install)
 * Build the binary for your current platform:
```
cd src/
go build -ldflags "-s -w" .
```
 * Build the binary for many platforms:
```
cd src/
go get -u && go mod tidy
bash build.sh  # Not running on Windows
```


## How to execute it directly (without building the binary in advance or using the pre-compiled)
 * [Install Go](https://go.dev/doc/install)
  * Execute:
```
cd src/
go run .
```
