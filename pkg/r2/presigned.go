package r2

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/okanay/go-template/pkg/utils"
)

// Input ve Output structları
type R2PresigInput struct {
	Filename     string `json:"filename" validate:"required"`
	ContentType  string `json:"contentType" validate:"required"`
	SizeInBytes  int64  `json:"sizeInBytes" validate:"required,max=10485760"`
	FileCategory string `json:"fileCategory"`
}

type R2PresignedOutput struct {
	PresignedURL string    `json:"presignedUrl"`
	UploadURL    string    `json:"uploadUrl"`
	ObjectKey    string    `json:"objectKey"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

func (r *R2) GeneratePresignedURL(ctx context.Context, input R2PresigInput) (*R2PresignedOutput, error) {
	ext := filepath.Ext(input.Filename)                // .docx
	nameRaw := strings.TrimSuffix(input.Filename, ext) // Okan Ay Vize
	safeName := utils.SanitizeFilename(nameRaw)        // okan-ay-vize
	finalFilename := fmt.Sprintf("%s-%s%s", safeName, utils.GenerateRandomString(8), ext)

	category := strings.TrimSpace(input.FileCategory)
	if category == "" {
		category = "general"
	}

	objectKey := path.Join(r.folderName, category, finalFilename)
	expiry := 5 * time.Minute
	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucketName),
		Key:           aws.String(objectKey),
		ContentType:   aws.String(input.ContentType),
		ContentLength: aws.Int64(input.SizeInBytes),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return nil, fmt.Errorf("[R2] :: failed to generate presigned URL: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", strings.TrimRight(r.publicURLBase, "/"), objectKey)

	return &R2PresignedOutput{
		PresignedURL: req.URL,
		UploadURL:    publicURL,
		ObjectKey:    objectKey,
		ExpiresAt:    time.Now().Add(expiry),
	}, nil
}
