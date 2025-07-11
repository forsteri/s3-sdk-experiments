package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	awsclient "s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
	"s3-uploader/internal/uploader"
)

func main() {
	// コマンドライン引数
	var (
		configFile = flag.String("config", "config.json", "設定ファイルのパス")
		source     = flag.String("source", "", "アップロード元のファイルまたはディレクトリ")
		bucket     = flag.String("bucket", "", "アップロード先のS3バケット")
		key        = flag.String("key", "", "S3キー（ファイル）またはプレフィックス（ディレクトリ）")
		recursive  = flag.Bool("recursive", false, "ディレクトリを再帰的にアップロード")
		dryRun     = flag.Bool("dry-run", false, "ドライランモード")
	)
	flag.Parse()

	// 必須引数のチェック
	if *source == "" {
		fmt.Fprintln(os.Stderr, "Error: -source is required")
		flag.Usage()
		os.Exit(1)
	}

	// 設定を読み込み
	cfg, err := models.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// ドライランモードの上書き
	if *dryRun {
		cfg.Options.DryRun = true
	}

	// ロガーをセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("ロガーの初期化に失敗: %v", err)
	}

	lgr := logger.GetLogger()
	lgr.Info("Upload test tool started",
		"source", *source,
		"dry_run", cfg.Options.DryRun,
	)

	// S3クライアントマネージャーを作成
	clientManager, err := awsclient.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	// アップローダーを作成
	upl := uploader.NewUploader(clientManager, cfg.Options)

	ctx := context.Background()

	// バケット名の決定
	uploadBucket := *bucket
	if uploadBucket == "" && len(cfg.UploadTasks) > 0 {
		uploadBucket = cfg.UploadTasks[0].Bucket
	}
	if uploadBucket == "" {
		lgr.Fatalf("バケット名が指定されていません")
	}

	// ファイルまたはディレクトリの情報を取得
	sourceInfo, err := os.Stat(*source)
	if err != nil {
		lgr.Fatalf("ソースの情報取得エラー: %v", err)
	}

	if sourceInfo.IsDir() {
		// ディレクトリのアップロード
		lgr.Info("Uploading directory",
			"path", *source,
			"bucket", uploadBucket,
			"prefix", *key,
			"recursive", *recursive,
		)

		results, err := upl.UploadDirectoryWithRetry(ctx, *source, uploadBucket, *key, *recursive)
		if err != nil {
			lgr.Fatalf("ディレクトリアップロードエラー: %v", err)
		}

		// 結果のサマリー
		var successCount, failureCount int
		var totalSize int64
		for _, result := range results {
			if result.Success {
				successCount++
				totalSize += result.Size
			} else {
				failureCount++
			}
		}

		lgr.Info("Upload completed",
			"total_files", len(results),
			"success", successCount,
			"failure", failureCount,
			"total_size", totalSize,
		)

		// 失敗したファイルの詳細
		if failureCount > 0 {
			fmt.Println("\n失敗したファイル:")
			for _, result := range results {
				if !result.Success {
					fmt.Printf("  - %s: %v\n", result.Source, result.Error)
				}
			}
		}
	} else {
		// 単一ファイルのアップロード
		uploadKey := *key
		if uploadKey == "" {
			uploadKey = sourceInfo.Name()
		}

		lgr.Info("Uploading file",
			"file", *source,
			"bucket", uploadBucket,
			"key", uploadKey,
		)

		result, err := upl.UploadFileWithRetry(ctx, *source, uploadBucket, uploadKey)
		if err != nil {
			lgr.Fatalf("ファイルアップロードエラー: %v", err)
		}

		if result.Success {
			lgr.Info("Upload successful",
				"size", result.Size,
				"key", result.Key,
			)
		}
	}
}
