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
		dryRun     = flag.Bool("dry-run", false, "ドライランモード")
		testMode   = flag.Bool("test", false, "テストモード（単一ファイルアップロードのテスト）")
	)
	flag.Parse()

	fmt.Println("🚀 S3 Uploader - Go version")
	fmt.Println("========================================")

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
		log.Fatalf("❌ ロガーの初期化に失敗: %v", err)
	}

	lgr := logger.GetLogger()
	lgr.Info("S3 Uploader initialized",
		"config_file", *configFile,
		"dry_run", cfg.Options.DryRun,
		"test_mode", *testMode,
	)

	// S3クライアントマネージャーを作成
	clientManager, err := awsclient.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	ctx := context.Background()

	if *testMode {
		// テストモード：単一ファイルアップロードのテスト
		testBucket := "s3-experiment-bucket-250615"
		testFile := "../test-data/sample_data.csv"
		key := "test-upload/sample_data.csv"

		lgr.Info("Running in test mode")

		// 接続テスト
		lgr.Info("Testing S3 connection...", "bucket", testBucket)
		if err := clientManager.TestConnection(ctx, testBucket); err != nil {
			lgr.Fatalf("S3接続テスト失敗: %v", err)
		}

		// テストファイルをアップロード
		lgr.Info("Uploading test file",
			"file", testFile,
			"bucket", testBucket,
			"key", key,
		)

		// メタデータを追加してアップロード
		metadata := map[string]string{
			"uploaded-by": "s3-uploader-go",
			"version":     "1.0.0",
			"mode":        "test",
		}

		err = clientManager.UploadFileWithMetadata(ctx, testBucket, key, testFile, metadata)
		if err != nil {
			lgr.Fatalf("アップロードエラー: %v", err)
		}

		// アップロードしたオブジェクトの存在確認
		exists, err := clientManager.ObjectExists(ctx, testBucket, key)
		if err != nil {
			lgr.Error("オブジェクト存在確認エラー", "error", err)
		} else if exists {
			lgr.Info("✅ アップロードしたオブジェクトの存在を確認しました")
		}

		fmt.Println("\n✅ テストモードが完了しました！")
	} else {
		// 通常モード：タスクランナーを実行
		lgr.Info("Starting task runner mode")

		// タスクランナーを作成
		runner := uploader.NewTaskRunner(clientManager, *cfg)

		// すべてのタスクを実行
		report, err := runner.RunAllTasks(ctx)
		if err != nil {
			lgr.Fatalf("タスク実行エラー: %v", err)
		}

		// レポートを表示
		runner.PrintReport(report)

		// 終了処理
		if report.FailedTasks > 0 {
			lgr.Error("⚠️  Some tasks failed", "failed_count", report.FailedTasks)
			os.Exit(1)
		} else {
			fmt.Println("\n✅ すべてのタスクが正常に完了しました！")
		}
	}
}
