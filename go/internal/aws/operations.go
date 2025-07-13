package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Operations S3操作に関するインターフェース
type S3Operations interface {
	UploadFile(ctx context.Context, bucket, key, filePath string) error
	UploadFileWithMetadata(ctx context.Context, bucket, key, filePath string, metadata map[string]string) error
	ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error)
	ObjectExists(ctx context.Context, bucket, key string) (bool, error)
	UploadFileMultipart(ctx context.Context, bucket, key, filePath string, chunkSize int64, metadata map[string]string) error
	UploadFileMultipartParallel(ctx context.Context, bucket, key, filePath string, chunkSize int64, numWorkers int, metadata map[string]string) error
}

// Ensure ClientManager implements S3Operations
var _ S3Operations = (*ClientManager)(nil)

// UploadFile ファイルをS3にアップロード
func (cm *ClientManager) UploadFile(ctx context.Context, bucket, key, filePath string) error {
	return cm.UploadFileWithMetadata(ctx, bucket, key, filePath, nil)
}

// UploadFileWithMetadata メタデータ付きでファイルをS3にアップロード
func (cm *ClientManager) UploadFileWithMetadata(ctx context.Context, bucket, key, filePath string, metadata map[string]string) error {
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// ファイル情報を取得
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	cm.logger.Debug("Uploading file",
		"file", filePath,
		"size", fileInfo.Size(),
		"bucket", bucket,
		"key", key,
	)

	// Content-Typeを推測
	contentType := cm.guessContentType(filePath)

	// PutObjectの入力を構築
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	// メタデータが指定されている場合
	if metadata != nil && len(metadata) > 0 {
		putInput.Metadata = metadata
	}

	// S3にアップロード
	_, err = cm.s3Client.PutObject(ctx, putInput)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	cm.logger.Info("File uploaded successfully",
		"file", filePath,
		"bucket", bucket,
		"key", key,
		"size", fileInfo.Size(),
	)

	return nil
}

// ListObjects S3バケット内のオブジェクトをリスト
func (cm *ClientManager) ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	cm.logger.Debug("Listing objects",
		"bucket", bucket,
		"prefix", prefix,
	)

	var objects []types.Object
	paginator := s3.NewListObjectsV2Paginator(cm.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		objects = append(objects, page.Contents...)
	}

	cm.logger.Debug("Objects listed",
		"count", len(objects),
	)

	return objects, nil
}

// ObjectExists S3オブジェクトの存在確認
func (cm *ClientManager) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	// まずHeadObjectを試す（権限があれば最も効率的）
	_, err := cm.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err == nil {
		return true, nil
	}

	// NotFoundエラーの場合は明確にfalseを返す
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}

	// 403 Forbiddenの場合は、ListObjectsで確認を試みる
	cm.logger.Debug("HeadObject failed with permission error, trying ListObjects", "error", err)
	
	// キーの親ディレクトリを取得
	prefix := key
	
	// ListObjectsで確認
	result, listErr := cm.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	
	if listErr != nil {
		// ListObjectsも失敗した場合は元のエラーを返す
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	
	// 結果を確認
	for _, obj := range result.Contents {
		if obj.Key != nil && *obj.Key == key {
			cm.logger.Debug("Object found via ListObjects", "key", key)
			return true, nil
		}
	}
	
	return false, nil
}

// UploadReader io.ReaderからS3にアップロード（内部使用）
func (cm *ClientManager) uploadReader(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	}

	if metadata != nil && len(metadata) > 0 {
		input.Metadata = metadata
	}

	_, err := cm.s3Client.PutObject(ctx, input)
	return err
}

// guessContentType ファイル拡張子からContent-Typeを推測
func (cm *ClientManager) guessContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz":
		return "application/gzip"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
