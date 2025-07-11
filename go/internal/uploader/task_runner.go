package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
)

// TaskRunner タスクの実行を管理する構造体
type TaskRunner struct {
	uploader *Uploader
	config   models.Config
	logger   *logger.Logger
}

// NewTaskRunner 新しいTaskRunnerを作成
func NewTaskRunner(client aws.S3Operations, config models.Config) *TaskRunner {
	lgr := logger.GetLogger()
	uploader := NewUploader(client, config.Options)
	
	return &TaskRunner{
		uploader: uploader,
		config:   config,
		logger:   lgr,
	}
}

// TaskResult タスク実行結果
type TaskResult struct {
	TaskName      string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	TotalFiles    int
	SuccessFiles  int
	FailedFiles   int
	SkippedFiles  int
	TotalBytes    int64
	UploadResults []UploadResult
	Error         error
}

// RunReport 実行レポート
type RunReport struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	TotalTasks   int
	SuccessTasks int
	FailedTasks  int
	SkippedTasks int
	TaskResults  []TaskResult
	DryRun       bool
}

// RunAllTasks すべてのタスクを実行
func (tr *TaskRunner) RunAllTasks(ctx context.Context) (*RunReport, error) {
	report := &RunReport{
		StartTime:   time.Now(),
		DryRun:      tr.config.Options.DryRun,
		TaskResults: make([]TaskResult, 0),
	}

	if tr.config.Options.DryRun {
		tr.logger.Info("🏃 Running in DRY RUN mode - no files will be uploaded")
	}

	tr.logger.Info("Starting task runner",
		"total_tasks", len(tr.config.UploadTasks),
		"dry_run", tr.config.Options.DryRun,
	)

	// 各タスクを実行
	for _, task := range tr.config.UploadTasks {
		// タスクが無効の場合はスキップ
		if !task.Enabled {
			tr.logger.Info("Skipping disabled task", "task", task.Name)
			report.SkippedTasks++
			continue
		}

		tr.logger.Info("📋 Starting task",
			"name", task.Name,
			"description", task.Description,
		)

		result := tr.runTask(ctx, task)
		report.TaskResults = append(report.TaskResults, result)

		if result.Error != nil {
			report.FailedTasks++
			tr.logger.Error("❌ Task failed",
				"task", task.Name,
				"error", result.Error,
			)
		} else {
			report.SuccessTasks++
			tr.logger.Info("✅ Task completed",
				"task", task.Name,
				"duration", result.Duration,
				"files", result.SuccessFiles,
				"bytes", result.TotalBytes,
			)
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	report.TotalTasks = len(tr.config.UploadTasks)

	return report, nil
}

// runTask 個別のタスクを実行
func (tr *TaskRunner) runTask(ctx context.Context, task models.UploadTask) TaskResult {
	result := TaskResult{
		TaskName:  task.Name,
		StartTime: time.Now(),
	}

	// ソースの存在確認
	sourceInfo, err := os.Stat(task.Source)
	if err != nil {
		result.Error = fmt.Errorf("source not found: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// ディレクトリかファイルかで処理を分岐
	if sourceInfo.IsDir() {
		// ディレクトリのアップロード
		recursive := task.Recursive // Recursiveは直接bool型

		// S3キープレフィックスを決定
		keyPrefix := ""
		if task.S3KeyPrefix != nil {
			keyPrefix = *task.S3KeyPrefix
		}

		tr.logger.Debug("Uploading directory",
			"source", task.Source,
			"bucket", task.Bucket,
			"prefix", keyPrefix,
			"recursive", recursive,
		)

		// リトライ機能を使ったディレクトリアップロード
		uploadResults, err := tr.uploader.UploadDirectoryWithRetry(ctx, task.Source, task.Bucket, keyPrefix, recursive)
		result.UploadResults = uploadResults
		result.Error = err

		// 結果を集計
		for _, ur := range uploadResults {
			result.TotalFiles++
			if ur.Success {
				if ur.SkippedReason != "" {
					result.SkippedFiles++
				} else {
					result.SuccessFiles++
					result.TotalBytes += ur.Size
				}
			} else {
				result.FailedFiles++
			}
		}
	} else {
		// 単一ファイルのアップロード
		var key string
		if task.S3Key != nil && *task.S3Key != "" {
			key = *task.S3Key
		} else {
			// S3キーが指定されていない場合はファイル名を使用
			key = filepath.Base(task.Source)
		}

		tr.logger.Debug("Uploading file",
			"source", task.Source,
			"bucket", task.Bucket,
			"key", key,
		)

		// リトライ機能を使ったファイルアップロード
		uploadResult, err := tr.uploader.UploadFileWithRetry(ctx, task.Source, task.Bucket, key)
		result.UploadResults = []UploadResult{*uploadResult}
		result.Error = err
		result.TotalFiles = 1

		if uploadResult.Success {
			if uploadResult.SkippedReason != "" {
				result.SkippedFiles = 1
			} else {
				result.SuccessFiles = 1
				result.TotalBytes = uploadResult.Size
			}
		} else {
			result.FailedFiles = 1
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// PrintReport レポートをコンソールに出力
func (tr *TaskRunner) PrintReport(report *RunReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 UPLOAD TASK RUNNER REPORT")
	fmt.Println(strings.Repeat("=", 60))

	if report.DryRun {
		fmt.Println("🏃 Mode: DRY RUN (no files were actually uploaded)")
	} else {
		fmt.Println("🏃 Mode: LIVE")
	}

	fmt.Printf("⏱️  Total Duration: %s\n", report.Duration.Round(time.Millisecond))
	fmt.Printf("📅 Start Time: %s\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("📅 End Time: %s\n", report.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// タスクサマリー
	fmt.Println("📋 Task Summary:")
	fmt.Printf("   Total Tasks: %d\n", report.TotalTasks)
	fmt.Printf("   ✅ Successful: %d\n", report.SuccessTasks)
	fmt.Printf("   ❌ Failed: %d\n", report.FailedTasks)
	fmt.Printf("   ⏭️  Skipped: %d\n", report.SkippedTasks)
	fmt.Println()

	// 各タスクの詳細
	fmt.Println("📝 Task Details:")
	for i, task := range report.TaskResults {
		fmt.Printf("\n%d. %s\n", i+1, task.TaskName)
		fmt.Printf("   Duration: %s\n", task.Duration.Round(time.Millisecond))
		fmt.Printf("   Files: %d total (%d success, %d failed, %d skipped)\n",
			task.TotalFiles, task.SuccessFiles, task.FailedFiles, task.SkippedFiles)
		fmt.Printf("   Total Size: %s\n", formatBytes(task.TotalBytes))

		if task.Error != nil {
			fmt.Printf("   ❌ Error: %v\n", task.Error)
		}

		// 失敗したファイルの詳細（最大5件）
		if task.FailedFiles > 0 {
			fmt.Println("   Failed Files:")
			count := 0
			for _, ur := range task.UploadResults {
				if !ur.Success && ur.Error != nil {
					fmt.Printf("     - %s: %v\n", ur.Source, ur.Error)
					count++
					if count >= 5 && task.FailedFiles > 5 {
						fmt.Printf("     ... and %d more\n", task.FailedFiles-5)
						break
					}
				}
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// formatBytes バイト数を人間が読みやすい形式に変換
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
