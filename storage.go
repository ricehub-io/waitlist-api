package main

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storer is the interface for uploading files to object storage.
type Storer interface {
	UploadFile(ctx context.Context, bucket, key string, body io.Reader, contentType string) error
}

// Storage is a wrapper around S3 object storage client
type Storage struct {
	client *s3.Client
}

func NewStorage(accessKey, secretKey, baseURL string) (*Storage, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(baseURL)
		o.UsePathStyle = true
	})

	return &Storage{client}, nil
}

func (s *Storage) UploadFile(
	ctx context.Context,
	bucket, key string,
	body io.Reader,
	contentType string,
) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}
