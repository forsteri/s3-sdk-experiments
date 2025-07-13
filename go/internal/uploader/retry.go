package uploader

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"s3-uploader/internal/fileutils"
)

// RetryConfig リトライ設定
type RetryConfig struct {
	MaxRetries int
	RetryDelay time.Duration
}

// UploadFileWithRetry リトライ機能付きでファイルをアップロード
func (u *Uploader) UploadFileWithRetry(ctx context.Context, filePath string, bucket string, key string) (*UploadResult, error) {
	maxRetries := u.uploadConfig.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr error
	var result *UploadResult

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// コンテキストがキャンセルされているかチェック
		select {
		case <-ctx.Done():
			return &UploadResult{
				Source:  filePath,
				Bucket:  bucket,
				Key:     key,
				Success: false,
				Error:   ctx.Err(),
			}, ctx.Err()
		default:
		}

		if attempt > 0 {
			// リトライの場合はログを出力
			u.logger.Info("Retrying upload",
				"file", filePath,
				"attempt", attempt+1,
				"max_retries", maxRetries+1,
			)

			// リトライ間隔（指数バックオフ）
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			time.Sleep(delay)
		}

		result, lastErr = u.UploadFile(ctx, filePath, bucket, key)
		if lastErr == nil {
			return result, nil
		}

		u.logger.Warning("Upload attempt failed",
			"file", filePath,
			"attempt", attempt+1,
			"error", lastErr,
		)
	}

	// すべてのリトライが失敗
	if result == nil {
		result = &UploadResult{
			Source:  filePath,
			Bucket:  bucket,
			Key:     key,
			Success: false,
		}
	}
	result.Error = fmt.Errorf("upload failed after %d attempts: %w", maxRetries+1, lastErr)
	return result, result.Error
}

// UploadDirectoryWithRetry リトライ機能付きでディレクトリをアップロード
func (u *Uploader) UploadDirectoryWithRetry(ctx context.Context, dirPath string, bucket string, keyPrefix string, recursive bool) ([]UploadResult, error) {
	// 並列アップロードが有効かつファイル数が多い場合は並列処理を使用
	if u.uploadConfig.ParallelUploads > 1 {
		// まずファイル数を確認
		files, err := u.scanner.ScanDirectory(dirPath, recursive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}
		
		// ファイル数が並列処理の閾値を超えている場合は並列処理を使用
		if len(files) > u.uploadConfig.ParallelUploads {
			u.logger.Info("Using parallel upload with retry for directory",
				"path", dirPath,
				"files", len(files),
				"workers", u.uploadConfig.ParallelUploads,
			)
			return u.UploadDirectoryParallel(ctx, dirPath, bucket, keyPrefix, recursive)
		}
	}
	
	// ファイル数が少ない場合は通常の順次処理
	return u.uploadDirectorySequentialWithRetry(ctx, dirPath, bucket, keyPrefix, recursive)
}

// uploadDirectorySequentialWithRetry 順次処理でディレクトリをアップロード（既存の処理）
func (u *Uploader) uploadDirectorySequentialWithRetry(ctx context.Context, dirPath string, bucket string, keyPrefix string, recursive bool) ([]UploadResult, error) {
	// ディレクトリをスキャン
	files, err := u.scanner.ScanDirectory(dirPath, recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	u.logger.Info("Directory scan completed",
		"path", dirPath,
		"files_found", len(files),
		"recursive", recursive,
	)

	// 各ファイルをアップロード（リトライ付き）
	results := make([]UploadResult, 0, len(files))
	for _, fileInfo := range files {
		// S3キーを生成
		key := filepath.Join(keyPrefix, fileInfo.RelativePath)
		// Windowsパスの場合、バックスラッシュをスラッシュに変換
		key = filepath.ToSlash(key)

		result, err := u.UploadFileWithRetry(ctx, fileInfo.Path, bucket, key)
		if err != nil {
			u.logger.Error("Failed to upload file after retries",
				"file", fileInfo.Path,
				"error", err,
			)
		}
		results = append(results, *result)
	}

	return results, nil
}

// CalculateTotalSize アップロード対象ファイルの合計サイズを計算
func (u *Uploader) CalculateTotalSize(files []fileutils.FileInfo) int64 {
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}
	return totalSize
}

// ShouldSkipFile ファイルをスキップすべきかチェック
func (u *Uploader) ShouldSkipFile(filePath string) (bool, string) {
	// 除外パターンはすでにスキャナーで処理されているので、
	// ここでは追加のチェックを行う（将来の拡張用）

	// 例: 0バイトファイルをスキップ
	info, err := u.scanner.GetFileInfo(filePath)
	if err == nil && info.Size == 0 && !u.uploadConfig.DryRun {
		return true, "empty file"
	}

	return false, ""
}
