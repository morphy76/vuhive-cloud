package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// Adapter implements outbound.StoragePort for AWS S3 and MinIO object storage.
type Adapter struct {
	bucket        string
	client        *s3.Client
	presignClient *s3.PresignClient
}

// NewAdapter constructs a new S3 / MinIO storage adapter from configuration.
func NewAdapter(ctx context.Context, cfg Config) (*Adapter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid s3 configuration: %w", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	presignClient := s3.NewPresignClient(client)

	return &Adapter{
		bucket:        cfg.Bucket,
		client:        client,
		presignClient: presignClient,
	}, nil
}

// NewAdapterWithClient initializes an Adapter with an existing S3 client and PresignClient (useful for testing).
func NewAdapterWithClient(bucket string, client *s3.Client, presignClient *s3.PresignClient) *Adapter {
	if presignClient == nil && client != nil {
		presignClient = s3.NewPresignClient(client)
	}
	return &Adapter{
		bucket:        bucket,
		client:        client,
		presignClient: presignClient,
	}
}

// EnsureBucket guarantees that the configured storage bucket exists, creating it if absent.
func (a *Adapter) EnsureBucket(ctx context.Context) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "StorageAdapter.EnsureBucket").Str("bucket", a.bucket).Logger()
	log.Debug().Msg("checking storage bucket existence")

	_, err := a.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(a.bucket),
	})
	if err == nil {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("storage bucket already exists")
		return nil
	}

	log.Debug().Msg("storage bucket not found; attempting creation")
	_, err = a.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(a.bucket),
	})
	if err != nil {
		var bAlreadyExists *s3types.BucketAlreadyExists
		var bAlreadyOwned *s3types.BucketAlreadyOwnedByYou
		if errors.As(err, &bAlreadyExists) || errors.As(err, &bAlreadyOwned) {
			log.Info().Dur("duration_ms", time.Since(start)).Msg("bucket already created by concurrent request")
			return nil
		}
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create storage bucket")
		return fmt.Errorf("failed to create storage bucket %q: %w", a.bucket, MapError(err))
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("storage bucket created successfully")
	return nil
}

// Upload stores content at the specified key.
func (a *Adapter) Upload(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return model.ErrEmptyS3Key
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.Upload").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Int64("size", size).
		Str("content_type", contentType).
		Logger()
	log.Debug().Msg("starting upload to storage")

	input := &s3.PutObjectInput{
		Bucket:      aws.String(a.bucket),
		Key:         aws.String(trimmedKey),
		Body:        content,
		ContentType: aws.String(contentType),
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := a.client.PutObject(ctx, input)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to upload object to storage")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed upload to storage")
	return nil
}

// Download retrieves the content stored at the specified key. Caller must close the returned ReadCloser.
func (a *Adapter) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, model.ErrEmptyS3Key
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.Download").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Logger()
	log.Debug().Msg("starting download from storage")

	output, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(trimmedKey),
	})
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to download object from storage")
		return nil, MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed download from storage")
	return output.Body, nil
}

// Delete removes the object stored at the specified key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return model.ErrEmptyS3Key
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.Delete").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Logger()
	log.Debug().Msg("starting object deletion from storage")

	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(trimmedKey),
	})
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete object from storage")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed object deletion from storage")
	return nil
}

// Exists checks whether an object exists at the specified key.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return false, model.ErrEmptyS3Key
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.Exists").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Logger()
	log.Debug().Msg("checking object existence in storage")

	_, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(trimmedKey),
	})
	if err != nil {
		mapped := MapError(err)
		if errors.Is(mapped, model.ErrNotFound) {
			log.Info().Dur("duration_ms", time.Since(start)).Msg("object does not exist")
			return false, nil
		}
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed checking object existence")
		return false, mapped
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("object exists in storage")
	return true, nil
}

// PresignDownload generates a presigned GET URL for downloading an object within the given lifetime.
func (a *Adapter) PresignDownload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", model.ErrEmptyS3Key
	}

	if lifetime <= 0 {
		lifetime = 15 * time.Minute
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.PresignDownload").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Dur("lifetime", lifetime).
		Logger()
	log.Debug().Msg("generating presigned download URL")

	req, err := a.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(trimmedKey),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to generate presigned download URL")
		return "", MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed presigned download URL generation")
	return req.URL, nil
}

// PresignUpload generates a presigned PUT URL for uploading an object within the given lifetime.
func (a *Adapter) PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	start := time.Now()
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", model.ErrEmptyS3Key
	}

	if lifetime <= 0 {
		lifetime = 15 * time.Minute
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.PresignUpload").
		Str("bucket", a.bucket).
		Str("key", trimmedKey).
		Dur("lifetime", lifetime).
		Logger()
	log.Debug().Msg("generating presigned upload URL")

	req, err := a.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(trimmedKey),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to generate presigned upload URL")
		return "", MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed presigned upload URL generation")
	return req.URL, nil
}

// ListObjects returns metadata for all objects matching the specified prefix.
func (a *Adapter) ListObjects(ctx context.Context, prefix string) ([]outbound.ObjectInfo, error) {
	start := time.Now()
	trimmedPrefix := strings.TrimSpace(prefix)

	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.ListObjects").
		Str("bucket", a.bucket).
		Str("prefix", trimmedPrefix).
		Logger()
	log.Debug().Msg("starting listing objects from storage")

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
	}
	if trimmedPrefix != "" {
		input.Prefix = aws.String(trimmedPrefix)
	}

	paginator := s3.NewListObjectsV2Paginator(a.client, input)
	var objects []outbound.ObjectInfo

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to list objects from storage")
			return nil, MapError(err)
		}
		for _, obj := range page.Contents {
			var key string
			if obj.Key != nil {
				key = *obj.Key
			}
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			var modTime time.Time
			if obj.LastModified != nil {
				modTime = *obj.LastModified
			}
			objects = append(objects, outbound.ObjectInfo{
				Key:          key,
				Size:         size,
				LastModified: modTime,
			})
		}
	}

	log.Info().Int("count", len(objects)).Dur("duration_ms", time.Since(start)).Msg("completed listing objects from storage")
	return objects, nil
}

// PutBucketLifecycleConfiguration applies native lifecycle expiration rules to the storage bucket.
func (a *Adapter) PutBucketLifecycleConfiguration(ctx context.Context, rules []outbound.LifecycleRule) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "StorageAdapter.PutBucketLifecycleConfiguration").
		Str("bucket", a.bucket).
		Int("rules_count", len(rules)).
		Logger()
	log.Debug().Msg("starting configuring bucket lifecycle")

	var s3Rules []s3types.LifecycleRule
	for _, r := range rules {
		status := s3types.ExpirationStatusDisabled
		if r.Enabled {
			status = s3types.ExpirationStatusEnabled
		}
		var days int32
		if r.ExpirationDays > 0 && r.ExpirationDays <= math.MaxInt32 {
			days = int32(r.ExpirationDays)
		} else if r.ExpirationDays > math.MaxInt32 {
			days = math.MaxInt32
		}
		rule := s3types.LifecycleRule{
			ID:     aws.String(r.ID),
			Status: status,
			Filter: &s3types.LifecycleRuleFilter{
				Prefix: aws.String(r.Prefix),
			},
			Expiration: &s3types.LifecycleExpiration{
				Days: aws.Int32(days),
			},
		}
		s3Rules = append(s3Rules, rule)
	}

	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(a.bucket),
		LifecycleConfiguration: &s3types.BucketLifecycleConfiguration{
			Rules: s3Rules,
		},
	}

	_, err := a.client.PutBucketLifecycleConfiguration(ctx, input)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed configuring bucket lifecycle")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed configuring bucket lifecycle")
	return nil
}

// Static compile-time interface assertion
var _ outbound.StoragePort = (*Adapter)(nil)

