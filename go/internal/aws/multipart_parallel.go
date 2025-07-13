package aws

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"s3-uploader/internal/progress"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ParallelMultipartUploader 並列マルチパートアップロードを管理する構造体
type ParallelMultipartUploader struct {
	manager     *MultipartUploadManager
	numWorkers  int
	chunkSize   int64
	file        *os.File
	fileSize    int64
	workerWg    sync.WaitGroup
	progressBar *progress.ProgressTracker
}

// PartJob 各ワーカーが処理するジョブ
type PartJob struct {
	PartNumber int32
	Offset     int64
	Size       int64
}

// NewParallelMultipartUploader 新しいParallelMultipartUploaderを作成
func NewParallelMultipartUploader(manager *MultipartUploadManager, file *os.File, fileSize int64, chunkSize int64, numWorkers int) *ParallelMultipartUploader {
	return &ParallelMultipartUploader{
		manager:    manager,
		numWorkers: numWorkers,
		chunkSize:  chunkSize,
		file:       file,
		fileSize:   fileSize,
	}
}

// Upload 並列でマルチパートアップロードを実行
func (pmu *ParallelMultipartUploader) Upload(ctx context.Context) error {
	// キャンセル可能なコンテキストを作成
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// パート数を計算
	totalParts := (pmu.fileSize + pmu.chunkSize - 1) / pmu.chunkSize

	// ジョブキューを作成
	jobQueue := make(chan PartJob, totalParts)
	errChan := make(chan error, pmu.numWorkers)

	// ワーカーを起動
	for i := 0; i < pmu.numWorkers; i++ {
		pmu.workerWg.Add(1)
		go pmu.worker(ctx, i, jobQueue, errChan)
	}

	// ジョブを生成
	var offset int64
	var partNumber int32
	for offset < pmu.fileSize {
		partSize := pmu.chunkSize
		if offset+partSize > pmu.fileSize {
			partSize = pmu.fileSize - offset
		}

		partNumber++
		jobQueue <- PartJob{
			PartNumber: partNumber,
			Offset:     offset,
			Size:       partSize,
		}

		offset += partSize
	}
	close(jobQueue)

	// ワーカーの完了を待つ
	go func() {
		pmu.workerWg.Wait()
		close(errChan)
	}()

	// エラーをチェック
	for err := range errChan {
		if err != nil {
			cancel() // エラーが発生したらキャンセル
			return err
		}
	}

	return nil
}

// worker アップロードワーカー
func (pmu *ParallelMultipartUploader) worker(ctx context.Context, workerID int, jobs <-chan PartJob, errChan chan<- error) {
	defer pmu.workerWg.Done()

	pmu.manager.client.logger.Debug("Multipart worker started", "worker_id", workerID)

	for job := range jobs {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}

		// ファイルの読み込み用バッファを作成
		buffer := make([]byte, job.Size)

		// ファイルからデータを読み込む（スレッドセーフ）
		n, err := pmu.file.ReadAt(buffer, job.Offset)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("worker %d: failed to read file part: %w", workerID, err)
			return
		}
		if int64(n) != job.Size {
			errChan <- fmt.Errorf("worker %d: read size mismatch: expected %d, got %d", workerID, job.Size, n)
			return
		}

		// パートをアップロード
		reader := bytes.NewReader(buffer[:n])
		result, err := pmu.manager.UploadPartWithNumber(ctx, reader, job.Size, job.PartNumber)
		if err != nil {
			errChan <- fmt.Errorf("worker %d: %w", workerID, err)
			return
		}

		pmu.manager.client.logger.Debug("Worker uploaded part",
			"worker_id", workerID,
			"part_number", result.PartNumber,
			"offset", job.Offset,
			"size", job.Size,
		)

		// 進捗を更新（プログレストラッカーが設定されている場合）
		if pmu.progressBar != nil {
			pmu.progressBar.IncrementProcessed(job.Size)
		}
	}

	pmu.manager.client.logger.Debug("Multipart worker finished", "worker_id", workerID)
}

// UploadPartWithNumber 指定されたパート番号でアップロード
func (mum *MultipartUploadManager) UploadPartWithNumber(ctx context.Context, reader io.Reader, partSize int64, partNumber int32) (*UploadPartResult, error) {
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

// UploadFileMultipartParallel ファイルを並列マルチパートでアップロード
func (cm *ClientManager) UploadFileMultipartParallel(ctx context.Context, bucket, key, filePath string, chunkSize int64, numWorkers int, metadata map[string]string) error {
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

	cm.logger.Info("Starting parallel multipart upload",
		"file", filePath,
		"size", fileSize,
		"chunk_size", chunkSize,
		"total_parts", (fileSize+chunkSize-1)/chunkSize,
		"workers", numWorkers,
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

	// 並列アップローダーを作成
	parallelUploader := NewParallelMultipartUploader(mum, file, fileSize, chunkSize, numWorkers)

	// アップロードを実行
	if err := parallelUploader.Upload(ctx); err != nil {
		uploadErr = err
		return uploadErr
	}

	// アップロードを完了
	if err := mum.CompleteMultipartUpload(ctx); err != nil {
		uploadErr = err
		return uploadErr
	}

	return nil
}
