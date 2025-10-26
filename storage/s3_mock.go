package storage

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MockS3Client is a mock implementation of S3Client for testing
type MockS3Client struct {
	// Objects stores mock S3 objects with their content and metadata
	Objects map[string]*MockS3Object
	// Buckets stores the list of buckets
	Buckets map[string]bool
	// Error to return from operations
	Err error
	// Track function calls
	HeadBucketCalled    bool
	PutObjectCalled     bool
	CreateBucketCalled  bool
	ListObjectsV2Called bool
	GetObjectCalled     bool
	HeadObjectCalled    bool
	// Store last call parameters
	LastBucket    string
	LastObjectKey string
	LastMetadata  map[string]string
}

// MockS3Object represents a mock S3 object with content and metadata
type MockS3Object struct {
	Key      string
	Content  string
	Metadata map[string]string
	Size     int64
}

// NewMockS3Client creates a new mock S3 client
func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		Objects: make(map[string]*MockS3Object),
		Buckets: make(map[string]bool),
	}
}

// HeadBucket mocks checking bucket existence
func (m *MockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	m.HeadBucketCalled = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}

	if m.Err != nil {
		return nil, m.Err
	}

	if params.Bucket != nil && m.Buckets[*params.Bucket] {
		return &s3.HeadBucketOutput{}, nil
	}

	return nil, &types.NoSuchBucket{}
}

// PutObject mocks uploading an object
func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.PutObjectCalled = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}
	if params.Key != nil {
		m.LastObjectKey = *params.Key
	}
	if params.Metadata != nil {
		m.LastMetadata = params.Metadata
	}

	if m.Err != nil {
		return nil, m.Err
	}

	// Read content from body if provided
	content := ""
	if params.Body != nil {
		data, err := io.ReadAll(params.Body)
		if err == nil {
			content = string(data)
		}
	}

	// Store the object
	if params.Key != nil {
		m.Objects[*params.Key] = &MockS3Object{
			Key:      *params.Key,
			Content:  content,
			Metadata: params.Metadata,
			Size:     int64(len(content)),
		}
	}

	return &s3.PutObjectOutput{}, nil
}

// CreateBucket mocks creating a bucket
func (m *MockS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	m.CreateBucketCalled = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}

	if m.Err != nil {
		return nil, m.Err
	}

	if params.Bucket != nil {
		m.Buckets[*params.Bucket] = true
	}

	return &s3.CreateBucketOutput{}, nil
}

// ListObjectsV2 mocks listing objects
func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	m.ListObjectsV2Called = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}

	if m.Err != nil {
		return nil, m.Err
	}

	// Filter objects by prefix if provided
	var contents []types.Object
	prefix := ""
	if params.Prefix != nil {
		prefix = *params.Prefix
	}

	for key, obj := range m.Objects {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			contents = append(contents, types.Object{
				Key:  aws.String(obj.Key),
				Size: aws.Int64(obj.Size),
			})
		}
	}

	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

// GetObject mocks retrieving an object
func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	m.GetObjectCalled = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}
	if params.Key != nil {
		m.LastObjectKey = *params.Key
	}

	if m.Err != nil {
		return nil, m.Err
	}

	if params.Key != nil {
		if obj, exists := m.Objects[*params.Key]; exists {
			return &s3.GetObjectOutput{
				Body:     io.NopCloser(strings.NewReader(obj.Content)),
				Metadata: obj.Metadata,
			}, nil
		}
		return nil, &types.NoSuchKey{}
	}

	return nil, &types.NoSuchKey{}
}

// HeadObject mocks retrieving object metadata
func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	m.HeadObjectCalled = true
	if params.Bucket != nil {
		m.LastBucket = *params.Bucket
	}
	if params.Key != nil {
		m.LastObjectKey = *params.Key
	}

	if m.Err != nil {
		return nil, m.Err
	}

	if params.Key != nil {
		if obj, exists := m.Objects[*params.Key]; exists {
			return &s3.HeadObjectOutput{
				Metadata:      obj.Metadata,
				ContentLength: aws.Int64(obj.Size),
			}, nil
		}
		return nil, &types.NoSuchKey{}
	}

	return nil, &types.NoSuchKey{}
}
