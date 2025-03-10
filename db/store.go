package db

import "io"

type OSS interface {
	UploadFile(bucketName string, objectName string, filePath string) error
	DownloadFile(bucketName string, objectName string, filePath string) error
	UpdateFolder(bucketName string, objectName string, folderPath string) error
	DownloadFolder(bucketName string, objectName string, folderPath string) error
	Upload(bucketName string, objectName string, reader io.Reader) error
	Download(bucketName string, objectName string, writer io.Writer) error
	Delete(bucketName string, objectName string) error
}

type Store struct {
	MongoDB
	// OSS
}

func NewStore(mongodb MongoDB, oss OSS) *Store {
	return &Store{
		MongoDB: mongodb,
		// OSS:     oss,
	}
}
