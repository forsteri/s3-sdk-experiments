package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"s3-uploader/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	fmt.Println("🚀 S3アップロードテスト開始...")

	// 1. 設定を読み込み（君が作った関数を使う）
	cfg, err := models.LoadFromFile("config.json")
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// 2. S3クライアントを作成
	s3Client, err := createS3Client(cfg.AWS)
	if err != nil {
		log.Fatalf("S3クライアント作成エラー: %v", err)
	}

	// 3. テストファイルをアップロード
	testFile := "../test-data/sample_data.csv"
	bucket := "s3-experiment-bucket-250615"
	key := "test-upload/sample_data.csv"

	fmt.Printf("📁 アップロード中: %s -> s3://%s/%s\n", testFile, bucket, key)

	err = uploadFile(s3Client, bucket, key, testFile)
	if err != nil {
		log.Fatalf("アップロードエラー: %v", err)
	}

	fmt.Println("✅ アップロード成功！")
}

// S3クライアントを作成
func createS3Client(awsConfig models.AWSConfig) (*s3.Client, error) {
	ctx := context.Background()

	// AWS設定をロード
	var cfg aws.Config
	var err error

	if awsConfig.Profile != nil && *awsConfig.Profile != "" {
		// プロファイルを使用
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsConfig.Region),
			config.WithSharedConfigProfile(*awsConfig.Profile),
		)
	} else {
		// デフォルト認証
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsConfig.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("AWS設定の読み込みエラー: %w", err)
	}

	// AssumeRoleが必要な場合
	if awsConfig.AssumeRole != nil {
		fmt.Println("🔐 AssumeRoleを実行中...")

		stsClient := sts.NewFromConfig(cfg)

		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(awsConfig.AssumeRole.RoleArn),
			RoleSessionName: aws.String(awsConfig.AssumeRole.SessionName),
			DurationSeconds: aws.Int32(int32(awsConfig.AssumeRole.DurationSeconds)),
		}

		if awsConfig.AssumeRole.ExternalID != nil {
			assumeRoleInput.ExternalId = awsConfig.AssumeRole.ExternalID
		}

		result, err := stsClient.AssumeRole(ctx, assumeRoleInput)
		if err != nil {
			return nil, fmt.Errorf("AssumeRoleエラー: %w", err)
		}

		// 一時的な認証情報を使って新しい設定を作成
		cfg.Credentials = credentials.NewStaticCredentialsProvider(
			*result.Credentials.AccessKeyId,
			*result.Credentials.SecretAccessKey,
			*result.Credentials.SessionToken,
		)

		fmt.Println("✅ AssumeRole成功！")
	}

	return s3.NewFromConfig(cfg), nil
}

// ファイルをアップロード
func uploadFile(client *s3.Client, bucket, key, filePath string) error {
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ファイルオープンエラー: %w", err)
	}
	defer file.Close()

	// ファイル情報を取得
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("ファイル情報取得エラー: %w", err)
	}

	fmt.Printf("📊 ファイルサイズ: %d bytes\n", fileInfo.Size())

	// S3にアップロード
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("S3アップロードエラー: %w", err)
	}

	return nil
}
