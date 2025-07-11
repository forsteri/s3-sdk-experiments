package aws

import (
	"context"
	"fmt"
	"time"

	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ClientManager S3クライアントを管理する構造体
type ClientManager struct {
	s3Client *s3.Client
	config   models.AWSConfig
	logger   *logger.Logger
}

// NewClientManager 新しいClientManagerを作成
func NewClientManager(awsConfig models.AWSConfig) (*ClientManager, error) {
	lgr := logger.GetLogger()
	
	manager := &ClientManager{
		config: awsConfig,
		logger: lgr,
	}
	
	// S3クライアントを初期化
	if err := manager.initializeClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}
	
	return manager, nil
}

// GetS3Client S3クライアントを取得
func (cm *ClientManager) GetS3Client() *s3.Client {
	return cm.s3Client
}

// initializeClient S3クライアントを初期化
func (cm *ClientManager) initializeClient() error {
	ctx := context.Background()
	
	cm.logger.Info("Initializing AWS client",
		"region", cm.config.Region,
		"has_profile", cm.config.Profile != nil,
		"has_assume_role", cm.config.AssumeRole != nil,
	)
	
	// AWS設定をロード
	cfg, err := cm.loadAWSConfig(ctx)
	if err != nil {
		return err
	}
	
	// AssumeRoleが必要な場合
	if cm.config.AssumeRole != nil {
		cfg, err = cm.assumeRole(ctx, cfg)
		if err != nil {
			return err
		}
	}
	
	// S3クライアントを作成
	cm.s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		// リトライ設定をカスタマイズ
		o.RetryMaxAttempts = 5
		o.RetryMode = aws.RetryModeAdaptive
	})
	
	cm.logger.Info("S3 client initialized successfully")
	return nil
}

// loadAWSConfig AWS設定をロード
func (cm *ClientManager) loadAWSConfig(ctx context.Context) (aws.Config, error) {
	var cfg aws.Config
	var err error
	
	// カスタムリトライ設定
	customRetry := retry.NewStandard(func(o *retry.StandardOptions) {
		o.MaxAttempts = 3
		o.MaxBackoff = 30 * time.Second
	})
	
	// ロード設定のオプション
	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cm.config.Region),
		config.WithRetryer(func() aws.Retryer {
			return customRetry
		}),
	}
	
	// プロファイルが指定されている場合
	if cm.config.Profile != nil && *cm.config.Profile != "" {
		cm.logger.Debug("Using AWS profile", "profile", *cm.config.Profile)
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(*cm.config.Profile))
	}
	
	// AWS設定をロード
	cfg, err = config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return cfg, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	return cfg, nil
}

// assumeRole AssumeRoleを実行して一時的な認証情報を取得
func (cm *ClientManager) assumeRole(ctx context.Context, cfg aws.Config) (aws.Config, error) {
	cm.logger.Info("Executing AssumeRole",
		"role_arn", cm.config.AssumeRole.RoleArn,
		"session_name", cm.config.AssumeRole.SessionName,
	)
	
	// STSクライアントを作成
	stsClient := sts.NewFromConfig(cfg)
	
	// AssumeRoleの入力を構築
	assumeRoleInput := &sts.AssumeRoleInput{
		RoleArn:         aws.String(cm.config.AssumeRole.RoleArn),
		RoleSessionName: aws.String(cm.config.AssumeRole.SessionName),
		DurationSeconds: aws.Int32(int32(cm.config.AssumeRole.DurationSeconds)),
	}
	
	// ExternalIDが設定されている場合
	if cm.config.AssumeRole.ExternalID != nil && *cm.config.AssumeRole.ExternalID != "" {
		assumeRoleInput.ExternalId = cm.config.AssumeRole.ExternalID
	}
	
	// AssumeRoleを実行
	result, err := stsClient.AssumeRole(ctx, assumeRoleInput)
	if err != nil {
		return cfg, fmt.Errorf("AssumeRole failed: %w", err)
	}
	
	// 一時的な認証情報を使って新しい設定を作成
	cfg.Credentials = credentials.NewStaticCredentialsProvider(
		*result.Credentials.AccessKeyId,
		*result.Credentials.SecretAccessKey,
		*result.Credentials.SessionToken,
	)
	
	cm.logger.Info("AssumeRole completed successfully",
		"expiration", result.Credentials.Expiration.Format(time.RFC3339),
	)
	
	return cfg, nil
}

// TestConnection S3接続をテスト
func (cm *ClientManager) TestConnection(ctx context.Context, bucket string) error {
	cm.logger.Debug("Testing S3 connection", "bucket", bucket)
	
	// バケットの存在確認
	_, err := cm.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}
	
	cm.logger.Info("S3 connection test successful", "bucket", bucket)
	return nil
}

