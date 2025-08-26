package storage

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	eve "eve.evalgo.org/common"
)

const MaxConcurrentUploads = 96

func lakeFsUploadFile(client *s3.Client, branch, bucket, objectKey, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Upload the file
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(branch + "/" + objectKey),
		Body:   file,
	})
	if err != nil {
		return err
	}

	eve.Logger.Info("✅ Uploaded file to bucket as", filePath, bucket, objectKey)
	return nil
}

func lakeFsEnsureBucketExists(client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	_, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return err
	}
	return nil
}

func LakeFSListObjects(url string, accessKey string, secretKey string, bucket string, branch string) {
	region := "us-east-1"
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	// _, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
	//     Bucket: aws.String(bucket),
	// })
	// if err != nil {
	//     eve.Logger.Fatal("Failed to access bucket", bucket, err)
	// }

	// filePath := "README.md"                              // path to the local file
	// objectKey := filepath.Base(filePath)                   // file name in the bucket

	// err = lakeFsEnsureBucketExists(client, bucket)
	// if err != nil {
	//     eve.Logger.Info("bucket failed: ", err)
	// }

	// err = lakeFsUploadFile(client, branch, bucket, objectKey, filePath)
	// if err != nil {
	//     eve.Logger.Info("Upload failed: ", err)
	// }

	// List objects
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(branch + "/"),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to list objects: ", err)
	}

	// eve.Logger.Info("Objects in bucket: ", bucket, " branch: ", branch, " output: ", output)
	for _, item := range output.Contents {
		eve.Logger.Info(*item.Key, " <=> ", *item.Size)
	}

}

func MinioListObjects(url string, accessKey string, secretKey string, bucket string) {
	region := "us-east-1"
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	_, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to access bucket", bucket, err)
	}

	// List objects
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to list objects: ", err)
	}

	// eve.Logger.Info("Objects in bucket: ", bucket, " branch: ", branch, " output: ", output)
	for _, item := range output.Contents {
		eve.Logger.Info(*item.Key, " <=> ", *item.Size)
	}
}

func MinioGetObject(url, accessKey, secretKey, bucket, remoteObject, localObject string) {
	region := "us-east-1"
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	_, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to access bucket", bucket, err)
	}

	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(remoteObject),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			eve.Logger.Fatal("Can't get object ", remoteObject, " from bucket ", bucket, ". No such key exists.\n")
			err = noKey
		} else {
			eve.Logger.Fatal("Couldn't get object ", bucket, ":", remoteObject, ". Here's why: ", err, "\n")
		}
		if err != nil {
			eve.Logger.Fatal("Failed to get object form bucket", bucket, err)
		}
	}
	defer result.Body.Close()
	file, err := os.Create(localObject)
	if err != nil {
		eve.Logger.Fatal("Couldn't create file ", localObject, ". Here's why: ", err, "\n")
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		eve.Logger.Fatal("Couldn't read object body from ", remoteObject, ". Here's why: ", err, "\n")
	}
	_, err = file.Write(body)
	if err != nil {
		eve.Logger.Fatal("Couldn't write object ", err, " \n")
	}
}

func HetznerListObjects(url, accessKey, secretKey, region, bucket string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	eve.Logger.Info(cfg)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	// _, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
	//     Bucket: aws.String(bucket),
	// })
	// if err != nil {
	//     eve.Logger.Fatal("Failed to access bucket", bucket, err)
	// }

	// List objects
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to list objects: ", err)
	}

	eve.Logger.Info("Objects in bucket: ", bucket, " region: ", region, " output: ", output.Contents)
	for _, item := range output.Contents {
		eve.Logger.Info(*item.Key, " <=> ", *item.Size)
	}
}

func HetznerGetObject(url, accessKey, secretKey, region, bucket, remoteObject, localObject string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	_, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to access bucket", bucket, err)
	}

	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(remoteObject),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			eve.Logger.Fatal("Can't get object ", remoteObject, " from bucket ", bucket, ". No such key exists.\n")
			err = noKey
		} else {
			eve.Logger.Fatal("Couldn't get object ", bucket, ":", remoteObject, ". Here's why: ", err, "\n")
		}
		if err != nil {
			eve.Logger.Fatal("Failed to get object form bucket", bucket, err)
		}
	}
	defer result.Body.Close()
	file, err := os.Create(localObject)
	if err != nil {
		eve.Logger.Fatal("Couldn't create file ", localObject, ". Here's why: ", err, "\n")
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		eve.Logger.Fatal("Couldn't read object body from ", remoteObject, ". Here's why: ", err, "\n")
	}
	_, err = file.Write(body)
	if err != nil {
		eve.Logger.Fatal("Couldn't write object ", err, " \n")
	}
}

func HetznerUploadFile(url, accessKey, secretKey, region, bucket, filePath, objectKey string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	// eve.Logger.Info(cfg)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}
	// Create an S3 clientuploadFile
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = &http.Client{}
	})
	uploader := manager.NewUploader(client)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	md5hash, err := CalculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate md5: %w", err)
	}
	defer file.Close()
	ctx := context.TODO()
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   file, // io.Reader,
		Metadata: map[string]string{
			"md5": md5hash, // This becomes x-amz-meta-md5 in S3
		},
	})
	if err != nil {
		return err
	}
	// eve.Logger.Info(output)
	return nil
}

func HetznerUploaderFile(ctx context.Context, uploader *manager.Uploader, bucket, filePath, objectKey string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	md5hash, err := CalculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate md5: %w", err)
	}
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   file, // io.Reader
		Metadata: map[string]string{
			"md5": md5hash, // This becomes x-amz-meta-md5 in S3
		},
	})
	if err != nil {
		return err
	}
	// eve.Logger.Info(output)
	return nil
}

func S3AwsListObjects(url, accessKey, secretKey, region, bucket string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           url,
					SigningRegion: region,
				}, nil
			})),
	)
	eve.Logger.Info(cfg)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = &http.Client{}
	})

	// Check if the bucket exists
	// _, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
	//     Bucket: aws.String(bucket),
	// })
	// if err != nil {
	//     eve.Logger.Fatal("Failed to access bucket", bucket, err)
	// }

	// List objects
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		eve.Logger.Fatal("Failed to list objects: ", err)
	}

	eve.Logger.Info("Objects in bucket: ", bucket, " region: ", region, " output: ", output.Contents)
	for _, item := range output.Contents {
		eve.Logger.Info(*item.Key, " <=> ", *item.Size)
	}
}

func GetAllLocalFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func HetznerUploadMultipleFiles(url, accessKey, secretKey, region, bucket, rootPath, objectKey string, syncToRemote bool) error {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true, // important for MinIO
				}, nil
			})),
	)
	// eve.Logger.Info(cfg)
	if err != nil {
		eve.Logger.Info("Failed to load configuration: ", err)
	}
	// Create an S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = &http.Client{}
	})
	uploader := manager.NewUploader(client)

	filePaths, err := GetAllLocalFiles(rootPath)
	if err != nil {
		return err
	}
	if syncToRemote {
		return HetznerSyncToRemote(ctx, client, uploader, bucket, filePaths, rootPath, objectKey)
	}
	return HetznerUploadToRemote(ctx, client, uploader, bucket, filePaths, rootPath, objectKey)
}

func HetznerUploadToRemote(ctx context.Context, client *s3.Client, uploader *manager.Uploader, bucket string, filePaths []string, rootPath, objectKey string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, MaxConcurrentUploads)

	for _, path := range filePaths {
		errChan <- errors.New("")
		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()
			defer func() { <-errChan }()

			relPath, err := filepath.Rel(rootPath, filePath)
			if err != nil {
				errChan <- fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
				return
			}

			// Convert path to S3 key format (Linux-style forward slashes)
			key := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

			if err := HetznerUploaderFile(ctx, uploader, bucket, filePath, objectKey+"/"+key); err != nil {
				errChan <- fmt.Errorf("failed to upload %s: %w", filePath, err)
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func HetznerSyncToRemote(ctx context.Context, client *s3.Client, uploader *manager.Uploader, bucket string, localFiles []string, rootPath, objectKey string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, MaxConcurrentUploads)

	for _, localPath := range localFiles {
		errChan <- errors.New("")
		wg.Add(1)

		go func(path string) {
			defer wg.Done()
			defer func() { <-errChan }()

			relPath, _ := filepath.Rel(rootPath, path)
			key := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

			// localInfo, err := os.Stat(path)
			// if err != nil {
			// 	errChan <- fmt.Errorf("failed to stat file %s: %w", path, err)
			// 	return
			// }

			localMD5, err := CalculateMD5(relPath)
			if err != nil {
				errChan <- fmt.Errorf("failed to md5 hash file %s: %w", path, err)
				return
			}

			head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(objectKey + "/" + key),
			})

			if err == nil {
				s3MD5 := head.Metadata["md5"] // S3 returns lowercase keys
				if s3MD5 == localMD5 {
					fmt.Printf("Skip (unchanged): %s\n", objectKey+"/"+key)
					return
				} else {
					fmt.Printf("Content differs: %s\n", objectKey+"/"+key)
					// If error from HeadObject or local is newer → upload
					err = HetznerUploaderFile(ctx, uploader, bucket, path, objectKey+"/"+key)
					if err != nil {
						errChan <- err
					}
				}
			} else {
				fmt.Printf("Content differs: %s\n", path)
				err := HetznerUploaderFile(ctx, uploader, bucket, path, objectKey+"/"+key)
				if err != nil {
					errChan <- err
				}
			}
		}(localPath)
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		return err
	}
	return nil
}

func CalculateMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
