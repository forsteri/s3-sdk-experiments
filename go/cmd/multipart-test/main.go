package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
)

func main() {
	// コマンドライン引数
	var (
		configFile      = flag.String("config", "config.json", "設定ファイルのパス")
		sourceFile      = flag.String("source", "", "アップロードするファイル (必須)")
		bucket          = flag.String("bucket", "", "S3バケット名 (必須)")
		key             = flag.String("key", "", "S3キー (必須)")
		region          = flag.String("region", "", "AWS リージョン (オプション、未指定時は設定ファイルから)")
		forceMultipart  = flag.Bool("multipart", false, "強制的にマルチパートアップロードを使用")
		chunkSizeMB     = flag.Int("chunk-size", 5, "チャンクサイズ (MB)")
		workers         = flag.Int("workers", 3, "並列ワーカー数")
		dryRun          = flag.Bool("dry-run", false, "ドライランモード")
		benchmark       = flag.Bool("benchmark", false, "ベンチマークモード（通常 vs マルチパート vs 並列マルチパートの比較）")
	)
	flag.Parse()

	// 必須パラメータのチェック
	if *sourceFile == "" || *bucket == "" || *key == "" {
		flag.Usage()
		log.Fatal("source, bucket, and key are required")
	}

	// ファイルの存在確認
	fileInfo, err := os.Stat(*sourceFile)
	if err != nil {
		log.Fatalf("Failed to access file: %v", err)
	}

	// 設定を読み込み
	cfg, err := models.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// コマンドラインでリージョンが指定されている場合は上書き
	if *region != "" {
		cfg.AWS.Region = *region
	}

	// ログセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	lgr := logger.GetLogger()
	lgr.Info("Multipart upload test started",
		"file", *sourceFile,
		"size", fileInfo.Size(),
		"bucket", *bucket,
		"key", *key,
		"region", cfg.AWS.Region,
	)

	// S3クライアントを作成
	clientManager, err := aws.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("Failed to create S3 client: %v", err)
	}

	ctx := context.Background()

	// チャンクサイズをバイトに変換
	chunkSize := int64(*chunkSizeMB) * 1024 * 1024

	// ベンチマークモード
	if *benchmark {
		runBenchmark(ctx, clientManager, *sourceFile, *bucket, *key, fileInfo.Size(), chunkSize, *workers)
		return
	}

	// ドライランモード
	if *dryRun {
		lgr.Info("DRY RUN: Would upload file",
			"method", getUploadMethod(fileInfo.Size(), *forceMultipart, cfg.Options.MultipartThreshold),
			"chunk_size", chunkSize,
			"workers", *workers,
		)
		return
	}

	// アップロード実行
	startTime := time.Now()

	if *forceMultipart || fileInfo.Size() >= cfg.Options.MultipartThreshold {
		// マルチパートアップロード
		if *workers > 1 {
			lgr.Info("Using parallel multipart upload",
				"workers", *workers,
				"chunk_size", chunkSize,
			)
			err = clientManager.UploadFileMultipartParallel(ctx, *bucket, *key, *sourceFile, chunkSize, *workers, nil)
		} else {
			lgr.Info("Using sequential multipart upload",
				"chunk_size", chunkSize,
			)
			err = clientManager.UploadFileMultipart(ctx, *bucket, *key, *sourceFile, chunkSize, nil)
		}
	} else {
		// 通常のアップロード
		lgr.Info("Using standard upload")
		err = clientManager.UploadFile(ctx, *bucket, *key, *sourceFile)
	}

	duration := time.Since(startTime)

	if err != nil {
		lgr.Fatalf("Upload failed: %v", err)
	}

	// 結果を表示
	throughput := float64(fileInfo.Size()) / duration.Seconds() / 1024 / 1024
	lgr.Info("Upload completed successfully",
		"duration", duration,
		"throughput_mbps", fmt.Sprintf("%.2f", throughput),
	)

	// アップロードしたオブジェクトの確認
	exists, err := clientManager.ObjectExists(ctx, *bucket, *key)
	if err != nil {
		lgr.Error("Failed to verify uploaded object", "error", err)
	} else if exists {
		lgr.Info("✅ Uploaded object verified")
	} else {
		lgr.Error("❌ Uploaded object not found")
	}
}

func getUploadMethod(fileSize int64, forceMultipart bool, threshold int64) string {
	if forceMultipart || fileSize >= threshold {
		return "multipart"
	}
	return "standard"
}

func runBenchmark(ctx context.Context, client *aws.ClientManager, sourceFile, bucket, key string, fileSize, chunkSize int64, workers int) {
	lgr := logger.GetLogger()

	lgr.Info("Running benchmark mode",
		"file_size", fileSize,
		"chunk_size", chunkSize,
	)

	// 結果を格納
	type Result struct {
		Method     string
		Duration   time.Duration
		Throughput float64
		Error      error
	}

	results := []Result{}

	// 1. 通常のアップロード（ファイルサイズが小さい場合のみ）
	if fileSize < 100*1024*1024 { // 100MB未満
		lgr.Info("Testing standard upload...")
		start := time.Now()
		err := client.UploadFile(ctx, bucket, key+"-standard", sourceFile)
		duration := time.Since(start)
		
		results = append(results, Result{
			Method:     "Standard",
			Duration:   duration,
			Throughput: float64(fileSize) / duration.Seconds() / 1024 / 1024,
			Error:      err,
		})
	}

	// 2. 順次マルチパートアップロード
	lgr.Info("Testing sequential multipart upload...")
	start := time.Now()
	err := client.UploadFileMultipart(ctx, bucket, key+"-multipart", sourceFile, chunkSize, nil)
	duration := time.Since(start)
	
	results = append(results, Result{
		Method:     "Multipart (Sequential)",
		Duration:   duration,
		Throughput: float64(fileSize) / duration.Seconds() / 1024 / 1024,
		Error:      err,
	})

	// 3. 並列マルチパートアップロード（異なるワーカー数でテスト）
	for _, w := range []int{2, 4, 8} {
		if w > workers {
			break
		}
		
		lgr.Info("Testing parallel multipart upload...", "workers", w)
		start := time.Now()
		err := client.UploadFileMultipartParallel(ctx, bucket, key+fmt.Sprintf("-parallel-%d", w), sourceFile, chunkSize, w, nil)
		duration := time.Since(start)
		
		results = append(results, Result{
			Method:     fmt.Sprintf("Multipart (Parallel %d workers)", w),
			Duration:   duration,
			Throughput: float64(fileSize) / duration.Seconds() / 1024 / 1024,
			Error:      err,
		})
	}

	// 結果を表示
	lgr.Info("Benchmark Results:")
	lgr.Info("==================")
	for _, r := range results {
		if r.Error != nil {
			lgr.Error("Method failed",
				"method", r.Method,
				"error", r.Error,
			)
		} else {
			lgr.Info("Result",
				"method", r.Method,
				"duration", r.Duration,
				"throughput_mbps", fmt.Sprintf("%.2f", r.Throughput),
			)
		}
	}
}
