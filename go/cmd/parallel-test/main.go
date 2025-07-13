package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
	"s3-uploader/internal/uploader"
)

func main() {
	// コマンドライン引数
	var (
		configFile   = flag.String("config", "config.json", "設定ファイルのパス")
		source       = flag.String("source", "", "アップロード元のファイルまたはディレクトリ")
		bucket       = flag.String("bucket", "s3-experiment-bucket-250615", "S3バケット名")
		key          = flag.String("key", "", "S3キー（ファイル）またはプレフィックス（ディレクトリ）")
		recursive    = flag.Bool("recursive", false, "ディレクトリを再帰的にアップロード")
		dryRun       = flag.Bool("dry-run", false, "ドライランモード")
		parallel     = flag.Bool("parallel", true, "並列アップロードを使用")
		workers      = flag.Int("workers", 0, "並列ワーカー数（0の場合は設定ファイルの値を使用）")
		benchmarkDir = flag.String("benchmark", "", "ベンチマーク用ディレクトリ（指定時は並列/順次の比較を実行）")
	)
	flag.Parse()

	if *source == "" && *benchmarkDir == "" {
		log.Fatal("❌ -source または -benchmark オプションを指定してください")
	}

	// 設定を読み込み
	cfg, err := models.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// コマンドラインオプションで設定を上書き
	if *dryRun {
		cfg.Options.DryRun = true
	}
	if *workers > 0 {
		cfg.Options.ParallelUploads = *workers
	}
	if !*parallel {
		cfg.Options.ParallelUploads = 1
	}

	// ロガーをセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("❌ ロガーの初期化に失敗: %v", err)
	}

	lgr := logger.GetLogger()

	// S3クライアントマネージャーを作成
	clientManager, err := aws.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	ctx := context.Background()

	// ベンチマークモード
	if *benchmarkDir != "" {
		runBenchmark(ctx, clientManager, cfg, *benchmarkDir, *bucket)
		return
	}

	// 通常のアップロードモード
	uploaderInstance := uploader.NewUploader(clientManager, cfg.Options)

	// ソースの存在確認
	sourceInfo, err := os.Stat(*source)
	if err != nil {
		lgr.Fatalf("❌ ソースが見つかりません: %v", err)
	}

	fmt.Println("🚀 Parallel Upload Test")
	fmt.Println("========================================")
	fmt.Printf("📁 Source: %s\n", *source)
	fmt.Printf("🪣 Bucket: %s\n", *bucket)
	fmt.Printf("🔑 Key/Prefix: %s\n", *key)
	fmt.Printf("👷 Workers: %d\n", cfg.Options.ParallelUploads)
	fmt.Printf("🏃 Dry Run: %v\n", cfg.Options.DryRun)
	fmt.Println("========================================")

	startTime := time.Now()

	if sourceInfo.IsDir() {
		// ディレクトリのアップロード
		if *key == "" {
			*key = "parallel-test/"
		}

		results, err := uploaderInstance.UploadDirectoryWithRetry(ctx, *source, *bucket, *key, *recursive)
		if err != nil {
			lgr.Fatalf("❌ ディレクトリアップロードエラー: %v", err)
		}

		// 結果の集計
		printResults(results, startTime)
	} else {
		// 単一ファイルのアップロード
		if *key == "" {
			*key = fmt.Sprintf("parallel-test/%s", sourceInfo.Name())
		}

		result, err := uploaderInstance.UploadFileWithRetry(ctx, *source, *bucket, *key)
		if err != nil {
			lgr.Fatalf("❌ ファイルアップロードエラー: %v", err)
		}

		printResults([]uploader.UploadResult{*result}, startTime)
	}
}

// runBenchmark 並列処理と順次処理のベンチマークを実行
func runBenchmark(ctx context.Context, clientManager aws.S3Operations, cfg *models.Config, source string, bucket string) {
	lgr := logger.GetLogger()

	fmt.Println("🏁 Benchmark Mode")
	fmt.Println("========================================")
	fmt.Printf("📁 Source: %s\n", source)
	fmt.Printf("🪣 Bucket: %s\n", bucket)
	fmt.Println("========================================")

	// 順次処理のテスト
	cfg.Options.ParallelUploads = 1
	sequentialUploader := uploader.NewUploader(clientManager, cfg.Options)

	fmt.Println("📊 Sequential Upload (1 worker)")
	fmt.Println("----------------------------------------")
	startTime := time.Now()
	results, err := sequentialUploader.UploadDirectoryWithRetry(ctx, source, bucket, "benchmark-sequential/", true)
	if err != nil {
		lgr.Error("Sequential upload failed", "error", err)
	} else {
		sequentialDuration := time.Since(startTime)
		printBenchmarkResult("Sequential", results, sequentialDuration)
	}

	// 並列処理のテスト（異なるワーカー数）
	workerCounts := []int{2, 4, 8, 16}
	for _, workers := range workerCounts {
		cfg.Options.ParallelUploads = workers
		parallelUploader := uploader.NewUploader(clientManager, cfg.Options)

		fmt.Printf("\n📊 Parallel Upload (%d workers)\n", workers)
		fmt.Println("----------------------------------------")
		startTime := time.Now()
		results, err := parallelUploader.UploadDirectoryWithRetry(ctx, source, bucket, fmt.Sprintf("benchmark-parallel-%d/", workers), true)
		if err != nil {
			lgr.Error("Parallel upload failed", "workers", workers, "error", err)
		} else {
			parallelDuration := time.Since(startTime)
			printBenchmarkResult(fmt.Sprintf("Parallel-%d", workers), results, parallelDuration)
		}
	}
}

// printResults アップロード結果を表示
func printResults(results []uploader.UploadResult, startTime time.Time) {
	duration := time.Since(startTime)
	totalFiles := len(results)
	successFiles := 0
	failedFiles := 0
	skippedFiles := 0
	totalBytes := int64(0)

	for _, result := range results {
		if result.Success {
			if result.SkippedReason != "" {
				skippedFiles++
			} else {
				successFiles++
				totalBytes += result.Size
			}
		} else {
			failedFiles++
		}
	}

	fmt.Println("\n📊 Upload Results")
	fmt.Println("========================================")
	fmt.Printf("⏱️  Duration: %s\n", duration.Round(time.Millisecond))
	fmt.Printf("📁 Total Files: %d\n", totalFiles)
	fmt.Printf("✅ Successful: %d\n", successFiles)
	fmt.Printf("❌ Failed: %d\n", failedFiles)
	fmt.Printf("⏭️  Skipped: %d\n", skippedFiles)
	fmt.Printf("📦 Total Size: %s\n", formatBytes(totalBytes))
	// 最小値を1ミリ秒として計算（ゼロ除算を防ぐ）
	seconds := math.Max(duration.Seconds(), 0.001)
	throughput := float64(totalBytes) / seconds
	fmt.Printf("🚀 Throughput: %s/s\n", formatBytes(int64(throughput)))

	// 失敗したファイルの詳細
	if failedFiles > 0 {
		fmt.Println("\n❌ Failed Files:")
		count := 0
		for _, result := range results {
			if !result.Success && result.Error != nil {
				fmt.Printf("  - %s: %v\n", result.Source, result.Error)
				count++
				if count >= 5 && failedFiles > 5 {
					fmt.Printf("  ... and %d more\n", failedFiles-5)
					break
				}
			}
		}
	}
}

// printBenchmarkResult ベンチマーク結果を表示
func printBenchmarkResult(mode string, results []uploader.UploadResult, duration time.Duration) {
	totalFiles := 0
	successFiles := 0
	totalBytes := int64(0)

	for _, result := range results {
		totalFiles++
		if result.Success && result.SkippedReason == "" {
			successFiles++
			totalBytes += result.Size
		}
	}

	fmt.Printf("✅ %s completed in %s\n", mode, duration.Round(time.Millisecond))

	// 最小値を1ミリ秒として計算（ゼロ除算を防ぐ）
	seconds := math.Max(duration.Seconds(), 0.001)

	throughput := float64(totalBytes) / seconds
	filesPerSecond := float64(successFiles) / seconds

	fmt.Printf("   Files: %d/%d (%.1f files/s)\n", successFiles, totalFiles, filesPerSecond)
	fmt.Printf("   Data: %s (%.1f MB/s)\n", formatBytes(totalBytes), throughput/1024/1024)
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
