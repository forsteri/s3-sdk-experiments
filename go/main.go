package main

import (
	"fmt"
	"log"

	"s3-uploader/internal/config"
)

func main() {
	fmt.Println("🚀 S3 Uploader - Go版の設定テスト")

	// 設定ファイルを読み込み
	cfg, err := config.LoadFromFile("config.json")
	if err != nil {
		log.Fatalf("❌ 設定の読み込みに失敗: %v", err)
	}

	fmt.Println("✅ 設定ファイルの読み込み成功！")
	fmt.Printf("  - AWS Region: %s\n", cfg.AWS.Region)
	fmt.Printf("  - Log Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  - Upload Tasks: %d個\n", len(cfg.UploadTasks))

	// AssumeRole設定の確認
	if cfg.AWS.AssumeRole != nil {
		fmt.Printf("  - AssumeRole: %s\n", cfg.AWS.AssumeRole.RoleArn)
	}

	// アップロードタスクの詳細
	for i, task := range cfg.UploadTasks {
		fmt.Printf("  - Task %d: %s (%s -> %s)\n", 
			i+1, task.Name, task.Source, task.Bucket)
	}

	fmt.Println("🎉 すべてのテストが完了しました！")
}
