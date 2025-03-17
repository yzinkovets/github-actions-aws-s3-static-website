package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	log "github.com/sirupsen/logrus"
	"github.com/yzinkovets/utils/env"
)

func main() {
	// Set log level
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true)

	env.SetLogger(log.StandardLogger())

	// Read inputs from environment variables
	accessKey := env.Must("INPUT_AWS_ACCESS_KEY_ID")
	secretKey := env.Must("INPUT_AWS_SECRET_ACCESS_KEY")
	region := env.Must("INPUT_AWS_REGION")
	domain := env.Must("INPUT_DOMAIN")
	indexDocument := env.GetDef("INPUT_INDEX_DOCUMENT", "index.html")
	errorDocument := env.GetDef("INPUT_ERROR_DOCUMENT", "index.html")
	actionOutputFile := env.Must("GITHUB_OUTPUT")

	// Bucket should be equal to domain name
	bucket := domain

	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	// Create a new AWS S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Check if bucket exists
	_, err = s3Client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err == nil {
		log.Infof("Bucket [%s] already exists", bucket)

		// To avoid replacement content of usual buckets (not static websites)
		// we have to check if bucket is configured as static website
		// If not, we should fail the action
		websiteConfig, err := s3Client.GetBucketWebsite(context.Background(), &s3.GetBucketWebsiteInput{
			Bucket: &bucket,
		})
		if err != nil {
			if getS3ErrorCode(err) == "NoSuchWebsiteConfiguration" {
				log.Fatal("Bucket already exists, but not configured as static website. Please remove it manually and try again")
			} else {
				log.Fatalf("Failed to get website configuration, %v", err)
			}
		}
		if websiteConfig == nil {
			log.Fatal("Bucket is already exists, but not configured as static website")
		}

		// Output website URL to let GitHub Actions continue the workflow
		outputWebsiteURL(actionOutputFile, bucket, region)

		return
	} else if getS3ErrorCode(err) != "NotFound" {
		log.Fatalf("Failed to check if bucket exists, %v", err)
	}

	input := s3.CreateBucketInput{
		Bucket: &bucket,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	}
	_, err = s3Client.CreateBucket(context.Background(), &input)
	if err != nil {
		log.Fatalf("Unable to create bucket, %v", err)
	}

	// Disable Block Public Access
	falseValue := false
	_, err = s3Client.PutPublicAccessBlock(context.Background(), &s3.PutPublicAccessBlockInput{
		Bucket: &bucket,
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       &falseValue,
			BlockPublicPolicy:     &falseValue,
			IgnorePublicAcls:      &falseValue,
			RestrictPublicBuckets: &falseValue,
		},
	})
	if err != nil {
		log.Fatalf("Failed to disable Block Public Access: %v", err)
	}

	bucketPolicyTemplate := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`
	bucketPolicy := fmt.Sprintf(bucketPolicyTemplate, bucket)

	_, err = s3Client.PutBucketPolicy(context.Background(), &s3.PutBucketPolicyInput{
		Bucket: &bucket,
		Policy: &bucketPolicy,
	})
	if err != nil {
		log.Fatalf("Unable to set bucket policy, %v", err)
	}
	log.Info("Bucket policy set successfully")

	// Set documents' configuration for static website
	_, err = s3Client.PutBucketWebsite(context.Background(), &s3.PutBucketWebsiteInput{
		Bucket: &bucket,
		WebsiteConfiguration: &types.WebsiteConfiguration{
			IndexDocument: &types.IndexDocument{
				Suffix: &indexDocument,
			},
			ErrorDocument: &types.ErrorDocument{
				Key: &errorDocument,
			},
		},
	})
	if err != nil {
		log.Fatalf("Unable to set website configuration, %v", err)
	}

	// Output website URL
	outputWebsiteURL(actionOutputFile, bucket, region)

	log.Info("Bucket website configuration set successfully")
}

// Helpers

func getS3ErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}
	return ""
}

func outputWebsiteURL(actionOutputFile, bucket, region string) {
	websiteEndpoint := fmt.Sprintf("%s.s3-website-%s.amazonaws.com", bucket, region)

	// Write the output variable to the file
	f, err := os.OpenFile(actionOutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening GITHUB_OUTPUT file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("website_url=%s\n", websiteEndpoint))
	if err != nil {
		log.Fatalf("Error writing to GITHUB_OUTPUT: %v", err)
	}
}
