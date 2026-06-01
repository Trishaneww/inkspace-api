package s3client

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	bucket    string
	s3        *s3.Client
	presigner *s3.PresignClient
}

func New(ctx context.Context, region, bucket string) (*Client, error) {
	if region == "" {
		return nil, fmt.Errorf("s3client: region is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("s3client: bucket is required")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("s3client: load aws config: %w", err)
	}

	s3svc := s3.NewFromConfig(cfg)
	return &Client{
		bucket:    bucket,
		s3:        s3svc,
		presigner: s3.NewPresignClient(s3svc),
	}, nil
}

func (c *Client) Bucket() string { return c.bucket }

func (c *Client) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3client: presign put: %w", err)
	}
	return req.URL, nil
}

func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3client: presign get: %w", err)
	}
	return req.URL, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3client: delete %q: %w", key, err)
	}
	return nil
}
