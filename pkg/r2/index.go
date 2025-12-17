package r2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2 struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
	folderName    string
	publicURLBase string
}

func NewR2Client(ctx context.Context, accountID, accessKeyID, accessKeySecret, bucketName, folderName, publicURLBase, endpoint string) (*R2, error) {
	// SDK konfigürasyonu
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			accessKeySecret,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("R2 config yüklenemedi: %w", err)
	}

	// S3 istemcisini oluştur
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // R2 için önemli
	})

	// Presign client'ı bir kere oluşturup reuse ediyoruz
	presignClient := s3.NewPresignClient(s3Client)

	return &R2{
		client:        s3Client,
		presignClient: presignClient,
		bucketName:    bucketName,
		folderName:    folderName,
		publicURLBase: publicURLBase,
	}, nil
}
