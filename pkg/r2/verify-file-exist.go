package r2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type R2FileMetadata struct {
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
}

func (r *R2) VerifyFileExists(ctx context.Context, objectKey string) (*R2FileMetadata, error) {
	output, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		var nfe *types.NotFound
		if errors.As(err, &nfe) {
			return nil, fmt.Errorf("[R2] :: File not found on R2: %s", objectKey)
		}

		return nil, fmt.Errorf("[R2] :: Error getting file information: %w", err)
	}

	// Pointer checks for safe assignment
	var size int64
	if output.ContentLength != nil {
		size = *output.ContentLength
	}

	var contentType string
	if output.ContentType != nil {
		contentType = *output.ContentType
	}

	var etag string
	if output.ETag != nil {
		etag = *output.ETag
	}

	var lastMod time.Time
	if output.LastModified != nil {
		lastMod = *output.LastModified
	}

	return &R2FileMetadata{
		Size:         size,
		ContentType:  contentType,
		LastModified: lastMod,
		ETag:         etag,
	}, nil
}
