package uploader

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"s3-uploader/internal/logger"
)

// UploadJob アップロードジョブを表す構造体
type UploadJob struct {
	FilePath string
	Bucket   string
	Key      string
	JobID    int
}

// ParallelUploader 並列アップロードを管理する構造体
type ParallelUploader struct {
	uploader      *Uploader
	logger        *logger.Logger
	numWorkers    int
	jobQueue      chan UploadJob
	resultQueue   chan UploadResult
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	uploadedCount atomic.Int64
	failedCount   atomic.Int64
	totalBytes    atomic.Int64
	stopped       atomic.Bool
}

// NewParallelUploader 新しいParallelUploaderを作成
func NewParallelUploader(uploader *Uploader, numWorkers int) *ParallelUploader {
	if numWorkers < 1 {
		numWorkers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ParallelUploader{
		uploader:    uploader,
		logger:      logger.GetLogger(),
		numWorkers:  numWorkers,
		jobQueue:    make(chan UploadJob, numWorkers*2), // バッファサイズはワーカー数の2倍
		resultQueue: make(chan UploadResult, numWorkers*2),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start ワーカープールを開始
func (pu *ParallelUploader) Start() {
	pu.logger.Info("Starting parallel uploader",
		"workers", pu.numWorkers,
	)

	// ワーカーを起動
	for i := 0; i < pu.numWorkers; i++ {
		pu.wg.Add(1)
		go pu.worker(i)
	}
}

// Stop ワーカープールを停止
func (pu *ParallelUploader) Stop() {
	// すでに停止している場合は何もしない
	if !pu.stopped.CompareAndSwap(false, true) {
		pu.logger.Debug("Parallel uploader already stopped")
		return
	}

	pu.logger.Info("Stopping parallel uploader")

	// ジョブキューをクローズ
	close(pu.jobQueue)

	// すべてのワーカーが終了するのを待つ
	pu.wg.Wait()

	// 結果キューをクローズ
	close(pu.resultQueue)

	// コンテキストをキャンセル
	pu.cancel()
}

// worker アップロードワーカー
func (pu *ParallelUploader) worker(id int) {
	defer pu.wg.Done()

	pu.logger.Debug("Worker started", "worker_id", id)

	for job := range pu.jobQueue {
		// コンテキストがキャンセルされていたら終了
		select {
		case <-pu.ctx.Done():
			pu.logger.Debug("Worker stopped due to context cancellation", "worker_id", id)
			return
		default:
		}

		// アップロード実行
		pu.logger.Debug("Worker processing job",
			"worker_id", id,
			"job_id", job.JobID,
			"file", job.FilePath,
		)

		startTime := time.Now()
		result, err := pu.uploader.UploadFileWithRetry(pu.ctx, job.FilePath, job.Bucket, job.Key)

		if err != nil {
			pu.logger.Error("Worker upload failed",
				"worker_id", id,
				"job_id", job.JobID,
				"file", job.FilePath,
				"error", err,
			)
			pu.failedCount.Add(1)
		} else if result.Success && result.SkippedReason == "" {
			pu.uploadedCount.Add(1)
			pu.totalBytes.Add(result.Size)

			pu.logger.Debug("Worker upload completed",
				"worker_id", id,
				"job_id", job.JobID,
				"file", job.FilePath,
				"duration", time.Since(startTime),
				"size", result.Size,
			)
		}

		// 結果をキューに送信
		select {
		case pu.resultQueue <- *result:
		case <-pu.ctx.Done():
			return
		}
	}

	pu.logger.Debug("Worker finished", "worker_id", id)
}

// SubmitJob ジョブをキューに追加
func (pu *ParallelUploader) SubmitJob(job UploadJob) error {
	select {
	case pu.jobQueue <- job:
		return nil
	case <-pu.ctx.Done():
		return fmt.Errorf("parallel uploader is stopped: context cancelled")
	}
}

// GetResults 結果キューのチャンネルを取得
func (pu *ParallelUploader) GetResults() <-chan UploadResult {
	return pu.resultQueue
}

// GetStats 現在の統計情報を取得
func (pu *ParallelUploader) GetStats() (uploaded, failed int64, totalBytes int64) {
	return pu.uploadedCount.Load(), pu.failedCount.Load(), pu.totalBytes.Load()
}

// UploadDirectoryParallel ディレクトリを並列でアップロード
func (u *Uploader) UploadDirectoryParallel(ctx context.Context, dirPath string, bucket string, keyPrefix string, recursive bool) ([]UploadResult, error) {
	// ディレクトリをスキャン
	files, err := u.scanner.ScanDirectory(dirPath, recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	u.logger.Info("Directory scan completed for parallel upload",
		"path", dirPath,
		"files_found", len(files),
		"recursive", recursive,
		"workers", u.uploadConfig.ParallelUploads,
	)

	// 並列アップローダーを作成
	parallelUploader := NewParallelUploader(u, u.uploadConfig.ParallelUploads)
	parallelUploader.Start()

	// 結果を収集するゴルーチン
	results := make([]UploadResult, 0, len(files))
	resultChan := make(chan []UploadResult)

	go func() {
		var collectedResults []UploadResult
		for result := range parallelUploader.GetResults() {
			collectedResults = append(collectedResults, result)
		}
		resultChan <- collectedResults
	}()

	// ジョブを投入
	for i, fileInfo := range files {
		// S3キーを生成
		key := generateS3Key(keyPrefix, fileInfo.RelativePath)

		job := UploadJob{
			FilePath: fileInfo.Path,
			Bucket:   bucket,
			Key:      key,
			JobID:    i,
		}

		if err := parallelUploader.SubmitJob(job); err != nil {
			u.logger.Error("Failed to submit job", "error", err)
			break
		}
	}

	// すべてのジョブが完了するのを待つ
	parallelUploader.Stop()

	// 結果を取得
	results = <-resultChan

	// 統計情報をログ出力
	uploaded, failed, totalBytes := parallelUploader.GetStats()
	u.logger.Info("Parallel upload completed",
		"total_files", len(files),
		"uploaded", uploaded,
		"failed", failed,
		"total_bytes", totalBytes,
	)

	return results, nil
}

// UploadFilesParallel 複数のファイルを並列でアップロード
func (u *Uploader) UploadFilesParallel(ctx context.Context, uploadJobs []UploadJob) ([]UploadResult, error) {
	if len(uploadJobs) == 0 {
		return []UploadResult{}, nil
	}

	u.logger.Info("Starting parallel file upload",
		"total_files", len(uploadJobs),
		"workers", u.uploadConfig.ParallelUploads,
	)

	// 並列アップローダーを作成
	parallelUploader := NewParallelUploader(u, u.uploadConfig.ParallelUploads)
	parallelUploader.Start()

	// 結果を収集するゴルーチン
	results := make([]UploadResult, 0, len(uploadJobs))
	resultChan := make(chan []UploadResult)

	go func() {
		var collectedResults []UploadResult
		for result := range parallelUploader.GetResults() {
			collectedResults = append(collectedResults, result)
		}
		resultChan <- collectedResults
	}()

	// ジョブを投入
	for _, job := range uploadJobs {
		if err := parallelUploader.SubmitJob(job); err != nil {
			u.logger.Error("Failed to submit job", "error", err)
			break
		}
	}

	// すべてのジョブが完了するのを待つ
	parallelUploader.Stop()

	// 結果を取得
	results = <-resultChan

	// 統計情報をログ出力
	uploaded, failed, totalBytes := parallelUploader.GetStats()
	u.logger.Info("Parallel upload completed",
		"total_files", len(uploadJobs),
		"uploaded", uploaded,
		"failed", failed,
		"total_bytes", totalBytes,
	)

	return results, nil
}
