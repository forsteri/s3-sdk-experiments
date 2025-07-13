package uploader

import (
	"context"
	"fmt"
	"path/filepath"

	"s3-uploader/internal/aws"
	"s3-uploader/internal/fileutils"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
)

// Uploader S3へのファイルアップロードを管理する構造体
type Uploader struct {
	client       aws.S3Operations
	scanner      *fileutils.FileScanner
	logger       *logger.Logger
	uploadConfig models.UploadOptions
}

// NewUploader 新しいUploaderを作成
func NewUploader(client aws.S3Operations, config models.UploadOptions) *Uploader {
	lgr := logger.GetLogger()
	scanner := fileutils.NewFileScanner(config.ExcludePatterns)
	
	return &Uploader{
		client:       client,
		scanner:      scanner,
		logger:       lgr,
		uploadConfig: config,
	}
}

// UploadResult アップロード結果を表す構造体
type UploadResult struct {
	Source       string // アップロード元のパス
	Bucket       string // アップロード先のバケット
	Key          string // S3のキー
	Size         int64  // ファイルサイズ
	Success      bool   // 成功したかどうか
	Error        error  // エラー（失敗時）
	SkippedReason string // スキップした理由（該当する場合）
}

// UploadFile 単一ファイルをS3にアップロード
func (u *Uploader) UploadFile(ctx context.Context, filePath string, bucket string, key string) (*UploadResult, error) {
	// ファイル情報を取得
	fileInfo, err := u.scanner.GetFileInfo(filePath)
	if err != nil {
		return &UploadResult{
			Source:  filePath,
			Bucket:  bucket,
			Key:     key,
			Success: false,
			Error:   fmt.Errorf("failed to get file info: %w", err),
		}, err
	}

	// ドライランモードの場合
	if u.uploadConfig.DryRun {
		u.logger.Info("DRY RUN: Would upload file",
			"source", filePath,
			"bucket", bucket,
			"key", key,
			"size", fileInfo.Size,
		)
		return &UploadResult{
			Source:  filePath,
			Bucket:  bucket,
			Key:     key,
			Size:    fileInfo.Size,
			Success: true,
			SkippedReason: "dry run mode",
		}, nil
	}

	// マルチパートアップロードの闾値をチェック
	if fileInfo.Size >= u.uploadConfig.MultipartThreshold {
		u.logger.Info("File size exceeds multipart threshold, using multipart upload",
			"file", filePath,
			"size", fileInfo.Size,
			"threshold", u.uploadConfig.MultipartThreshold,
		)
		
		// 並列アップロードが有効な場合
		if u.uploadConfig.ParallelUploads > 1 {
			err = u.client.UploadFileMultipartParallel(ctx, bucket, key, filePath, 
				u.uploadConfig.MultipartChunksize, u.uploadConfig.ParallelUploads, nil)
		} else {
			err = u.client.UploadFileMultipart(ctx, bucket, key, filePath, 
				u.uploadConfig.MultipartChunksize, nil)
		}
		
		if err != nil {
			return &UploadResult{
				Source:  filePath,
				Bucket:  bucket,
				Key:     key,
				Size:    fileInfo.Size,
				Success: false,
				Error:   err,
			}, err
		}
		
		u.logger.Info("Multipart upload completed successfully",
			"source", filePath,
			"bucket", bucket,
			"key", key,
			"size", fileInfo.Size,
		)
		
		return &UploadResult{
			Source:  filePath,
			Bucket:  bucket,
			Key:     key,
			Size:    fileInfo.Size,
			Success: true,
		}, nil
	}

	// 通常のアップロード
	u.logger.Debug("Uploading file",
		"source", filePath,
		"bucket", bucket,
		"key", key,
		"size", fileInfo.Size,
	)

	err = u.client.UploadFile(ctx, bucket, key, filePath)
	if err != nil {
		return &UploadResult{
			Source:  filePath,
			Bucket:  bucket,
			Key:     key,
			Size:    fileInfo.Size,
			Success: false,
			Error:   err,
		}, err
	}

	u.logger.Info("File uploaded successfully",
		"source", filePath,
		"bucket", bucket,
		"key", key,
		"size", fileInfo.Size,
	)

	return &UploadResult{
		Source:  filePath,
		Bucket:  bucket,
		Key:     key,
		Size:    fileInfo.Size,
		Success: true,
	}, nil
}

// UploadDirectory ディレクトリ内のファイルをS3にアップロード
func (u *Uploader) UploadDirectory(ctx context.Context, dirPath string, bucket string, keyPrefix string, recursive bool) ([]UploadResult, error) {
	// 並列アップロードが有効かつワーカー数が2以上の場合は並列処理を使用
	if u.uploadConfig.ParallelUploads > 1 {
		return u.UploadDirectoryParallel(ctx, dirPath, bucket, keyPrefix, recursive)
	}

	// 以下は順次処理
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

	// 各ファイルをアップロード
	results := make([]UploadResult, 0, len(files))
	for _, fileInfo := range files {
		// S3キーを生成
		key := filepath.Join(keyPrefix, fileInfo.RelativePath)
		// Windowsパスの場合、バックスラッシュをスラッシュに変換
		key = filepath.ToSlash(key)

		result, err := u.UploadFile(ctx, fileInfo.Path, bucket, key)
		if err != nil {
			u.logger.Error("Failed to upload file",
				"file", fileInfo.Path,
				"error", err,
			)
			// エラーが発生してもアップロードを続行
		}
		results = append(results, *result)
	}

	return results, nil
}

// GenerateS3Key アップロード元のパスとタスク設定からS3キーを生成
func (u *Uploader) GenerateS3Key(sourcePath string, task models.UploadTask) (string, error) {
	// ファイル情報を取得
	info, err := u.scanner.GetFileInfo(sourcePath)
	if err != nil {
		// ディレクトリの可能性があるので、エラーは無視
		info = &fileutils.FileInfo{
			Path:         sourcePath,
			RelativePath: filepath.Base(sourcePath),
		}
	}

	// S3キーを決定
	var key string
	if task.S3Key != nil && *task.S3Key != "" {
		// 明示的にキーが指定されている場合
		key = *task.S3Key
	} else if task.S3KeyPrefix != nil && *task.S3KeyPrefix != "" {
		// プレフィックスが指定されている場合
		key = filepath.Join(*task.S3KeyPrefix, info.RelativePath)
	} else {
		// デフォルトはファイル名をそのまま使用
		key = info.RelativePath
	}

	// Windowsパスの場合、バックスラッシュをスラッシュに変換
	key = filepath.ToSlash(key)

	return key, nil
}
