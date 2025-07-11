package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestConnectionMode 接続テストのモード
type TestConnectionMode string

const (
	// TestModeHeadBucket HeadBucket APIを使用（ListBucket権限が必要）
	TestModeHeadBucket TestConnectionMode = "head_bucket"
	// TestModePutObject 実際に小さなオブジェクトをアップロード（PutObject権限のみ必要）
	TestModePutObject TestConnectionMode = "put_object"
	// TestModeAuto 自動選択（HeadBucketを試し、失敗したらPutObjectを試す）
	TestModeAuto TestConnectionMode = "auto"
)

// TestConnectionOptions 接続テストのオプション
type TestConnectionOptions struct {
	Mode          TestConnectionMode
	TestKeyPrefix string // TestModePutObjectで使用するキーのプレフィックス
	// CleanupTestObj フィールドは将来の実装のために予約
	// 現在はDeleteObject権限が必要なため実装していない
	// CleanupTestObj bool
}

// DefaultTestConnectionOptions デフォルトのテストオプション
func DefaultTestConnectionOptions() TestConnectionOptions {
	return TestConnectionOptions{
		Mode:          TestModeAuto,
		TestKeyPrefix: "_test-objects/s3-uploader/", // より明示的なプレフィックス
	}
}

// TestConnectionWithOptions オプション付きでS3接続をテスト
func (cm *ClientManager) TestConnectionWithOptions(ctx context.Context, bucket string, opts TestConnectionOptions) error {
	cm.logger.Debug("Testing S3 connection", "bucket", bucket, "mode", opts.Mode)

	switch opts.Mode {
	case TestModeHeadBucket:
		return cm.testWithHeadBucket(ctx, bucket)
	case TestModePutObject:
		return cm.testWithPutObject(ctx, bucket, opts.TestKeyPrefix)
	case TestModeAuto:
		// まずHeadBucketを試す
		if err := cm.testWithHeadBucket(ctx, bucket); err == nil {
			return nil
		}
		// 失敗したらPutObjectを試す
		cm.logger.Debug("HeadBucket failed, trying PutObject test")
		return cm.testWithPutObject(ctx, bucket, opts.TestKeyPrefix)
	default:
		return fmt.Errorf("unknown test mode: %s", opts.Mode)
	}
}

// testWithHeadBucket HeadBucket APIを使った接続テスト
func (cm *ClientManager) testWithHeadBucket(ctx context.Context, bucket string) error {
	_, err := cm.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}
	
	cm.logger.Info("S3 connection test successful (HeadBucket)", "bucket", bucket)
	return nil
}

// testWithPutObject 実際にオブジェクトをアップロードして接続テスト
func (cm *ClientManager) testWithPutObject(ctx context.Context, bucket string, prefix string) error {
	// テスト用のキーを生成（ナノ秒を使用して衝突を防ぐ）
	testKey := fmt.Sprintf("%sconnection-test-%d.txt", prefix, time.Now().UnixNano())
	testContent := fmt.Sprintf("S3 connection test at %s", time.Now().Format(time.RFC3339))
	
	cm.logger.Debug("Testing with PutObject", "key", testKey)
	
	// テストオブジェクトをアップロード
	_, err := cm.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(testKey),
		Body:        strings.NewReader(testContent),
		ContentType: aws.String("text/plain"),
		Metadata: map[string]string{
			"purpose":   "connection-test",
			"timestamp": time.Now().Format(time.RFC3339),
			"tool":      "s3-uploader-go",
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to upload test object to bucket %s: %w", bucket, err)
	}
	
	cm.logger.Info("S3 connection test successful (PutObject)", 
		"bucket", bucket,
		"test_key", testKey,
	)
	
	// 注意: テストオブジェクトは自動削除されません
	// 必要に応じて、S3のライフサイクルポリシーで
	// _test-objects/ プレフィックスのオブジェクトを定期的に削除することを推奨
	
	return nil
}

