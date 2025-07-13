package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
)

func main() {
	var (
		source = flag.String("source", "", "アップロード元のファイルまたはディレクトリ")
		bucket = flag.String("bucket", "s3-experiment-bucket-250615", "S3バケット名")
		key    = flag.String("key", "", "S3キー")
		debug  = flag.Bool("debug", false, "デバッグモードを有効化")
		trace  = flag.Bool("trace", false, "処理の詳細をトレース")
	)
	flag.Parse()

	if *source == "" || *key == "" {
		fmt.Println("Usage: debug-test -source <file/dir> -key <s3-key>")
		os.Exit(1)
	}

	// デバッグ用の簡易ロガー設定
	_, err := logger.Setup(models.LoggingConfig{
		Level:  "DEBUG",
		Format: "%(asctime)s - %(levelname)s - %(message)s",
	})
	if err != nil {
		fmt.Printf("Logger setup failed: %v\n", err)
		os.Exit(1)
	}

	lgr := logger.GetLogger()

	if *debug {
		lgr.Info("=== デバッグモード有効 ===")
		lgr.Info("処理の流れを詳細に出力します")
	}

	// 1. ファイルスキャンフェーズ
	if *trace {
		lgr.Info("[TRACE] Phase 1: ファイルスキャン開始")
		time.Sleep(500 * time.Millisecond) // 処理の流れを見やすくする
	}

	// ここで実際のアップロード処理の一部を呼び出して動作を確認
	// 例: scanner.ScanDirectory() の動作を確認

	if *trace {
		lgr.Info("[TRACE] Phase 2: S3クライアント初期化")
		time.Sleep(500 * time.Millisecond)
	}

	// S3クライアントの初期化処理

	if *trace {
		lgr.Info("[TRACE] Phase 3: アップロード処理")
		lgr.Info(fmt.Sprintf("  - ソース: %s", *source))
		lgr.Info(fmt.Sprintf("  - バケット: %s", *bucket))
		lgr.Info(fmt.Sprintf("  - キー: %s", *key))
	}

	// アップロード処理の呼び出し

	lgr.Info("=== 処理完了 ===")
}
