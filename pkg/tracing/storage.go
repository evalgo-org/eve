package tracing

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// uploadToS3 uploads data to S3 storage
func (t *Tracer) uploadToS3(ctx context.Context, correlationID, operationID, filename string, data []byte) error {
	if t.config.S3Client == nil {
		return fmt.Errorf("S3 client not configured")
	}

	// Construct S3 key: {correlation_id}/{operation_id}/{filename}
	key := fmt.Sprintf("%s/%s/%s", correlationID, operationID, filename)

	// Upload to S3
	_, err := t.config.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(t.config.S3Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	return err
}

// uploadLogsToS3 uploads logs to S3
func (t *Tracer) UploadLogs(ctx context.Context, correlationID, operationID string, logs []byte) error {
	return t.uploadToS3(ctx, correlationID, operationID, "logs.txt", logs)
}

// uploadArtifactToS3 uploads build artifacts to S3
func (t *Tracer) UploadArtifact(ctx context.Context, correlationID, operationID, artifactName string, data []byte) error {
	if t.config.S3Client == nil {
		return fmt.Errorf("S3 client not configured")
	}

	key := fmt.Sprintf("%s/%s/artifacts/%s", correlationID, operationID, artifactName)

	_, err := t.config.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(t.config.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})

	return err
}

// downloadFromS3 retrieves data from S3
func (t *Tracer) downloadFromS3(ctx context.Context, correlationID, operationID, filename string) ([]byte, error) {
	if t.config.S3Client == nil {
		return nil, fmt.Errorf("S3 client not configured")
	}

	key := fmt.Sprintf("%s/%s/%s", correlationID, operationID, filename)

	result, err := t.config.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(t.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	// Read response body
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GetRequest retrieves the full request JSON-LD from S3
func (t *Tracer) GetRequest(ctx context.Context, correlationID, operationID string) ([]byte, error) {
	return t.downloadFromS3(ctx, correlationID, operationID, "request.json")
}

// GetResponse retrieves the full response JSON-LD from S3
func (t *Tracer) GetResponse(ctx context.Context, correlationID, operationID string) ([]byte, error) {
	return t.downloadFromS3(ctx, correlationID, operationID, "response.json")
}

// GetLogs retrieves execution logs from S3
func (t *Tracer) GetLogs(ctx context.Context, correlationID, operationID string) ([]byte, error) {
	return t.downloadFromS3(ctx, correlationID, operationID, "logs.txt")
}
