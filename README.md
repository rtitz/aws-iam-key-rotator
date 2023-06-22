# AWS IAM Key Rotator

This is an AWS IAM Key Rotator created in Go. (You do not need to use the Go language, pre-compiled binaries available here.)
It can rotate AWS IAM Keys of one or multiple AWS CLI profiles.
Key rotation can be done in sequence or in a parallel mode, which is much faster if you have many AWS CLI profiles.

Regular key rotation is a best practice for security reasons.


## Requirements for the AWS IAM Key Rotator
 * Ensure that the AWS CLI is installed and configured! (https://docs.aws.amazon.com/cli/)
 * AWS CLI configuration (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
 * An IAM user with access key (and secret access key)
 * Your IAM user should have the following permissions (can be a separate attached IAM policy):
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "iam:CreateAccessKey",
                "iam:UpdateAccessKey",
                "iam:DeleteAccessKey",
                "iam:ListAccessKeys"
            ],
            "Resource": "arn:aws:iam::*:user/${aws:username}",
            "Effect": "Allow"
        }
    ]
}
```


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


---
## [Build it on your own from source](doc/build.md)
