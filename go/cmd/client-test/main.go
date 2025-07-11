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
)

func main() {
	// コマンドライン引数
	var (
		configFile = flag.String("config", "config.json", "設定ファイルのパス")
		bucket     = flag.String("bucket", "", "テストに使用するS3バケット名")
	)
	flag.Parse()

	// 設定を読み込み
	cfg, err := models.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// ロガーをセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("ロガーの初期化に失敗: %v", err)
	}

	lgr := logger.GetLogger()
	lgr.Info("AWS S3 Client Test Tool started")

	// S3クライアントマネージャーを作成
	clientManager, err := awsclient.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	ctx := context.Background()

	// バケット名が指定されていない場合は設定から取得
	testBucket := *bucket
	if testBucket == "" {
		if len(cfg.UploadTasks) > 0 {
			testBucket = cfg.UploadTasks[0].Bucket
		} else {
			lgr.Fatalf("テスト用のバケット名が指定されていません")
		}
	}

	// 1. 接続テスト
	fmt.Println("\n=== S3接続テスト ===")
	lgr.Info("Testing S3 connection", "bucket", testBucket)
	if err := clientManager.TestConnection(ctx, testBucket); err != nil {
		lgr.Fatalf("S3接続テスト失敗: %v", err)
	}
	fmt.Println("✅ 接続テスト成功")

	// 2. オブジェクト一覧取得テスト
	fmt.Println("\n=== オブジェクト一覧取得テスト ===")
	objects, err := clientManager.ListObjects(ctx, testBucket, "")
	if err != nil {
		lgr.Error("オブジェクト一覧取得失敗", "error", err)
	} else {
		lgr.Info("オブジェクト一覧取得成功", "count", len(objects))
		if len(objects) > 0 {
			fmt.Printf("最初の5個のオブジェクト:\n")
			for i, obj := range objects {
				if i >= 5 {
					break
				}
				fmt.Printf("  - %s (Size: %d bytes)\n", *obj.Key, obj.Size)
			}
		}
	}

	// 3. テストファイルのアップロード
	fmt.Println("\n=== ファイルアップロードテスト ===")
	
	// テスト用の一時ファイルを作成
	tempFile, err := os.CreateTemp("", "s3-test-*.txt")
	if err != nil {
		lgr.Fatalf("一時ファイル作成エラー: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	testContent := "This is a test file for S3 client testing\n"
	if _, err := tempFile.WriteString(testContent); err != nil {
		lgr.Fatalf("ファイル書き込みエラー: %v", err)
	}
	tempFile.Close()

	testKey := "test/client-test.txt"
	metadata := map[string]string{
		"test-type": "client-test",
		"uploaded-by": "s3-client-test",
	}

	lgr.Info("Uploading test file", "key", testKey)
	if err := clientManager.UploadFileWithMetadata(ctx, testBucket, testKey, tempFile.Name(), metadata); err != nil {
		lgr.Fatalf("アップロードエラー: %v", err)
	}
	fmt.Println("✅ アップロード成功")

	// 4. 存在確認テスト
	fmt.Println("\n=== オブジェクト存在確認テスト ===")
	exists, err := clientManager.ObjectExists(ctx, testBucket, testKey)
	if err != nil {
		lgr.Error("存在確認エラー", "error", err)
	} else if exists {
		fmt.Println("✅ アップロードしたファイルの存在を確認")
	} else {
		fmt.Println("❌ アップロードしたファイルが見つかりません")
	}

	fmt.Println("\n=== すべてのテストが完了しました ===")
}
