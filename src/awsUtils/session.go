package awsUtils

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func CreateAwsSession(ctx context.Context, awsProfile, awsRegion string) (aws.Config, error) {
	if len(awsProfile) == 0 {
		awsProfile = "default"
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(awsRegion),
		config.WithSharedConfigProfile(awsProfile),
	)
	if err != nil {
		log.Fatal(err)
	}
	return cfg, err
}
