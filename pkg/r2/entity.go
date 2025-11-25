package r2

import "time"

type UploadInput struct {
	Filename     string `json:"filename" validate:"required,min=5,file_ext=jpg png webp jpeg svg pdf docx xls xlsx mp4 mov avi"`
	ContentType  string `json:"contentType" validate:"required"`
	SizeInBytes  int64  `json:"sizeInBytes" validate:"required,max=10485760"`
	FileCategory string `json:"fileCategory"`
}

type UploadOutput struct {
	PresignedURL string    `json:"presignedUrl"`
	UploadURL    string    `json:"uploadUrl"`
	ObjectKey    string    `json:"objectKey"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type FileMetadata struct {
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
}
