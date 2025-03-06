package db

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioOSS struct {
	client *minio.Client
}

func NewMinioOSS(endpoint, accessKeyID, secretAccessKey string, useSSL bool) (*MinioOSS, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioOSS{client: client}, nil
}

// UpdateFolder 将指定文件夹压缩并上传到 MinIO
func (m *MinioOSS) UpdateFolder(bucketName string, objectName string, folderPath string) error {
	// 创建一个缓冲区来存储压缩后的文件
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// 遍历文件夹并将文件添加到压缩包中
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 在压缩包中创建文件
		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return err
		}
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// 将文件内容复制到压缩包中
		_, err = io.Copy(zipFile, file)
		return err
	})
	if err != nil {
		return err
	}

	// 关闭压缩包
	err = zipWriter.Close()
	if err != nil {
		return err
	}

	// 上传压缩包到 MinIO
	_, err = m.client.PutObject(context.Background(), bucketName, objectName, &buf, int64(buf.Len()), minio.PutObjectOptions{})
	return err
}

// DownloadFolder 从 MinIO 下载压缩包并解压到指定文件夹
func (m *MinioOSS) DownloadFolder(bucketName string, objectName string, folderPath string) error {
	// 从 MinIO 下载压缩包
	obj, err := m.client.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer obj.Close()

	// 读取压缩包内容
	zipData, err := ioutil.ReadAll(obj)
	if err != nil {
		return err
	}

	// 创建一个 Reader 来读取压缩包
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return err
	}

	// 解压文件到指定文件夹
	for _, zipFile := range zipReader.File {
		filePath := filepath.Join(folderPath, zipFile.Name)

		// 创建文件夹
		if zipFile.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// 创建文件
		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			return err
		}
		outFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		// 解压文件内容
		zipFileReader, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer zipFileReader.Close()

		_, err = io.Copy(outFile, zipFileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MinioOSS) UploadFile(bucketName, objectName, filePath string) error {
	_, err := m.client.FPutObject(context.Background(), bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
	}
	return err
}

func (m *MinioOSS) DownloadFile(bucketName, objectName, filePath string) error {
	err := m.client.FGetObject(context.Background(), bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("Failed to download file: %v", err)
	}
	return err
}

func (m *MinioOSS) Upload(bucketName, objectName string, reader io.Reader) error {
	_, err := m.client.PutObject(context.Background(), bucketName, objectName, reader, -1, minio.PutObjectOptions{})
	if err != nil {
		log.Printf("Failed to upload object: %v", err)
	}
	return err
}

func (m *MinioOSS) Download(bucketName, objectName string, writer io.Writer) error {
	object, err := m.client.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("Failed to get object: %v", err)
		return err
	}
	_, err = io.Copy(writer, object)
	if err != nil {
		log.Printf("Failed to copy object to writer: %v", err)
	}
	return err
}

func (m *MinioOSS) Delete(bucketName, objectName string) error {
	err := m.client.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("Failed to delete object: %v", err)
	}
	return err
}
