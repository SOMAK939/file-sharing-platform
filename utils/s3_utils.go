package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DeleteFromS3 removes a file from AWS S3
func DeleteFromS3(fileURL string) error {
	bucketName := os.Getenv("AWS_S3_BUCKET_NAME")
	key := extractFileNameFromURL(fileURL)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	_, err = s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %v", err)
	}

	log.Printf("🗑️ Deleted file from S3: %s\n", fileURL)
	return nil
}

// Extracts the S3 object key from the URL
func extractFileNameFromURL(fileURL string) string {
	parts := strings.Split(fileURL, "/")
	return parts[len(parts)-1] // Extract last part as file name
}
