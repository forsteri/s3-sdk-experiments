package main

import (
	"context"
	"fmt"
	"log"

	awsclient "s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
)

func main() {
	fmt.Println("🚀 S3アップロードテスト開始...")

	// 1. 設定を読み込み
	cfg, err := models.LoadFromFile("config.json")
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// 2. ロガーをセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("❌ ロガーの初期化に失敗: %v", err)
	}

	// ロガーを取得
	lgr := logger.GetLogger()
	lgr.Info("S3 Uploader initialized")
	lgr.Info("設定ファイルの読み込み成功",
		"region", cfg.AWS.Region,
		"tasks", len(cfg.UploadTasks),
	)

	// 3. S3クライアントマネージャーを作成
	clientManager, err := awsclient.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	// 4. 接続テスト
	ctx := context.Background()
	testBucket := "datalake-poc-raw-891376985958"

	lgr.Info("S3接続テストを実行中...", "bucket", testBucket)
	if err := clientManager.TestConnection(ctx, testBucket); err != nil {
		lgr.Fatalf("S3接続テスト失敗: %v", err)
	}

	// 5. テストファイルをアップロード
	testFile := "../test-data/sample_data.csv"
	key := "test-upload/sample_data.csv"

	lgr.Info("ファイルアップロードを開始",
		"file", testFile,
		"bucket", testBucket,
		"key", key,
	)

	// メタデータを追加してアップロード
	metadata := map[string]string{
		"uploaded-by": "s3-uploader-go",
		"version":     "1.0.0",
	}

	err = clientManager.UploadFileWithMetadata(ctx, testBucket, key, testFile, metadata)
	if err != nil {
		lgr.Fatalf("アップロードエラー: %v", err)
	}

	// 6. アップロードしたオブジェクトの存在確認
	exists, err := clientManager.ObjectExists(ctx, testBucket, key)
	if err != nil {
		lgr.Error("オブジェクト存在確認エラー", "error", err)
	} else if exists {
		lgr.Info("アップロードしたオブジェクトの存在を確認しました")
	}

	// 7. オブジェクト一覧を取得してみる
	objects, err := clientManager.ListObjects(ctx, testBucket, "test-upload/")
	if err != nil {
		lgr.Error("オブジェクト一覧取得エラー", "error", err)
	} else {
		lgr.Info("オブジェクト一覧",
			"prefix", "test-upload/",
			"count", len(objects),
		)
		for _, obj := range objects {
			lgr.Debug("Object found",
				"key", *obj.Key,
				"size", obj.Size,
				"modified", obj.LastModified,
			)
		}
	}

	fmt.Println("✅ すべてのテストが完了しました！")
}
