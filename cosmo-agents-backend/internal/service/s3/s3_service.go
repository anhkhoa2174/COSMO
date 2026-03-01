package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Service handles S3 file operations
type S3Service struct {
	client *s3.Client
}

// S3Config holds S3 configuration
type S3Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	EndpointURL     string // Optional: for localstack/minio
}

// NewS3Service creates a new S3 service
func NewS3Service(cfg S3Config) (*S3Service, error) {
	var awsCfg aws.Config
	var err error

	// Load AWS configuration
	if cfg.EndpointURL != "" {
		// Custom endpoint (localstack/minio)
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.EndpointURL,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true, // Force path-style for custom endpoints (MinIO/LocalStack)
			}, nil
		})

		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
			config.WithEndpointResolverWithOptions(customResolver),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Standard AWS
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style addressing for MinIO compatibility
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.UsePathStyle = true // Required for MinIO
		}
	})

	return &S3Service{
		client: client,
	}, nil
}

// UploadFile uploads a file to S3
func (s *S3Service) UploadFile(ctx context.Context, bucket, key string, data io.Reader, contentType string, metadata map[string]string) error {
	// Read all data into memory (for small files)
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, data); err != nil {
		return fmt.Errorf("failed to read file data: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

// GetObject retrieves an object from S3
func (s *S3Service) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// ListObjects lists objects in S3 with a given prefix
func (s *S3Service) ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return result.Contents, nil
}

// GeneratePresignedURL generates a presigned URL for downloading an object
func (s *S3Service) GeneratePresignedURL(ctx context.Context, bucket, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// DeleteObject deletes an object from S3
func (s *S3Service) DeleteObject(ctx context.Context, bucket, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
