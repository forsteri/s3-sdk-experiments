package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MultipartUploadManager マルチパートアップロードを管理する構造体
type MultipartUploadManager struct {
	client     *ClientManager
	uploadID   string
	bucket     string
	key        string
	parts      []types.CompletedPart
	partsMutex sync.Mutex
	partNumber int32
}

// UploadPartResult パートアップロードの結果
type UploadPartResult struct {
	PartNumber int32
	ETag       string
	Error      error
}

// CreateMultipartUpload マルチパートアップロードを開始
func (cm *ClientManager) CreateMultipartUpload(ctx context.Context, bucket, key, contentType string, metadata map[string]string) (*MultipartUploadManager, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	if metadata != nil && len(metadata) > 0 {
		input.Metadata = metadata
	}

	result, err := cm.s3Client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	cm.logger.Info("Multipart upload created",
		"bucket", bucket,
		"key", key,
		"upload_id", *result.UploadId,
	)

	return &MultipartUploadManager{
		client:   cm,
		uploadID: *result.UploadId,
		bucket:   bucket,
		key:      key,
		parts:    make([]types.CompletedPart, 0),
	}, nil
}

// UploadPart 単一パートをアップロード
func (mum *MultipartUploadManager) UploadPart(ctx context.Context, reader io.Reader, partSize int64) (*UploadPartResult, error) {
	partNumber := atomic.AddInt32(&mum.partNumber, 1)

	input := &s3.UploadPartInput{
		Bucket:        aws.String(mum.bucket),
		Key:           aws.String(mum.key),
		UploadId:      aws.String(mum.uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          reader,
		ContentLength: aws.Int64(partSize),
	}

	result, err := mum.client.s3Client.UploadPart(ctx, input)
	if err != nil {
		return &UploadPartResult{
			PartNumber: partNumber,
			Error:      fmt.Errorf("failed to upload part %d: %w", partNumber, err),
		}, err
	}

	// 完了したパートを記録
	mum.partsMutex.Lock()
	mum.parts = append(mum.parts, types.CompletedPart{
		ETag:       result.ETag,
		PartNumber: aws.Int32(partNumber),
	})
	mum.partsMutex.Unlock()

	mum.client.logger.Debug("Part uploaded",
		"part_number", partNumber,
		"size", partSize,
		"etag", *result.ETag,
	)

	return &UploadPartResult{
		PartNumber: partNumber,
		ETag:       *result.ETag,
	}, nil
}

// CompleteMultipartUpload マルチパートアップロードを完了
func (mum *MultipartUploadManager) CompleteMultipartUpload(ctx context.Context) error {
	// パート番号でソート
	mum.partsMutex.Lock()
	sort.Slice(mum.parts, func(i, j int) bool {
		return *mum.parts[i].PartNumber < *mum.parts[j].PartNumber
	})
	parts := mum.parts
	mum.partsMutex.Unlock()

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(mum.bucket),
		Key:      aws.String(mum.key),
		UploadId: aws.String(mum.uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	}

	_, err := mum.client.s3Client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	mum.client.logger.Info("Multipart upload completed",
		"bucket", mum.bucket,
		"key", mum.key,
		"upload_id", mum.uploadID,
		"total_parts", len(parts),
	)

	return nil
}

// AbortMultipartUpload マルチパートアップロードを中止
func (mum *MultipartUploadManager) AbortMultipartUpload(ctx context.Context) error {
	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(mum.bucket),
		Key:      aws.String(mum.key),
		UploadId: aws.String(mum.uploadID),
	}

	_, err := mum.client.s3Client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	mum.client.logger.Info("Multipart upload aborted",
		"bucket", mum.bucket,
		"key", mum.key,
		"upload_id", mum.uploadID,
	)

	return nil
}

// UploadFileMultipart ファイルをマルチパートでアップロード
func (cm *ClientManager) UploadFileMultipart(ctx context.Context, bucket, key, filePath string, chunkSize int64, metadata map[string]string) error {
	// S3の最小パートサイズは5MB（最後のパート以外）
	const minPartSize = 5 * 1024 * 1024 // 5MB
	if chunkSize < minPartSize {
		cm.logger.Warn("Chunk size is below S3 minimum, adjusting to 5MB", 
			"requested", chunkSize, 
			"adjusted", minPartSize)
		chunkSize = minPartSize
	}
	
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// ファイル情報を取得
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()
	
	// Content-Typeを推測
	contentType := cm.guessContentType(filePath)

	cm.logger.Info("Starting multipart upload",
		"file", filePath,
		"size", fileSize,
		"chunk_size", chunkSize,
		"total_parts", (fileSize+chunkSize-1)/chunkSize,
	)

	// マルチパートアップロードを開始
	mum, err := cm.CreateMultipartUpload(ctx, bucket, key, contentType, metadata)
	if err != nil {
		return err
	}

	// エラー時はアップロードを中止
	var uploadErr error
	defer func() {
		if uploadErr != nil {
			if abortErr := mum.AbortMultipartUpload(context.Background()); abortErr != nil {
				cm.logger.Error("Failed to abort multipart upload", "error", abortErr)
			}
		}
	}()

	// 各パートをアップロード
	var offset int64
	for offset < fileSize {
		// パートサイズを計算（最後のパートは小さい可能性がある）
		partSize := chunkSize
		if offset+partSize > fileSize {
			partSize = fileSize - offset
		}

		// ファイルの現在位置にシーク
		_, err := file.Seek(offset, 0)
		if err != nil {
			uploadErr = fmt.Errorf("failed to seek file: %w", err)
			return uploadErr
		}

		// パートをアップロード（LimitReaderで指定サイズのみ読み込む）
		result, err := mum.UploadPart(ctx, io.LimitReader(file, partSize), partSize)
		if err != nil {
			uploadErr = err
			return uploadErr
		}

		cm.logger.Debug("Part uploaded",
			"part_number", result.PartNumber,
			"offset", offset,
			"size", partSize,
		)

		offset += partSize
	}

	// アップロードを完了
	if err := mum.CompleteMultipartUpload(ctx); err != nil {
		uploadErr = err
		return uploadErr
	}

	return nil
}
