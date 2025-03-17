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

type Store interface {
	Set(place, key string, value []byte) error //禁止相同Key
	Get(place, key string) ([]byte, error)
	Delete(place, key string) error
	Update(place, key string, value []byte) error
}
