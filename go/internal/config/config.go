package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// LoggingConfig ログ設定を表す構造体
type LoggingConfig struct {
	Level  string  `json:"level"`
	Format string  `json:"format"`
	File   *string `json:"file"` // ポインタを使ってnull許可
}

// AssumeRoleConfig AWS AssumeRoleの設定を表す構造体
type AssumeRoleConfig struct {
	RoleArn         string  `json:"role_arn"`
	SessionName     string  `json:"session_name"`
	ExternalID      *string `json:"external_id"` // nullableなのでポインタ
	DurationSeconds int     `json:"duration_seconds"`
}

// Validate AssumeRole設定の検証を行う
func (a *AssumeRoleConfig) Validate() error {
	// ARNパターンの検証
	arnPattern := `^arn:aws:iam::[0-9]{12}:role\/[a-zA-Z0-9+=,.@_-]+$`
	if matched, _ := regexp.MatchString(arnPattern, a.RoleArn); !matched {
		return fmt.Errorf("invalid role_arn format: %s", a.RoleArn)
	}

	// セッション名の検証
	if a.SessionName == "" {
		return fmt.Errorf("session_name cannot be empty")
	}

	sessionNamePattern := `^[a-zA-Z0-9_.-]{2,64}$`
	if matched, _ := regexp.MatchString(sessionNamePattern, a.SessionName); !matched {
		return fmt.Errorf("invalid session_name: %s", a.SessionName)
	}

	// 期間の検証
	if a.DurationSeconds < 900 || a.DurationSeconds > 43200 {
		return fmt.Errorf("invalid duration_seconds: %d", a.DurationSeconds)
	}

	return nil
}

// AWSConfig AWS関連の設定を表す構造体
type AWSConfig struct {
	Region     string            `json:"region"`
	Profile    *string           `json:"profile"`     // nullableなのでポインタ
	AssumeRole *AssumeRoleConfig `json:"assume_role"` // nullableなのでポインタ
}

// UploadOptions アップロード設定オプションを表す構造体
type UploadOptions struct {
	DryRun             bool     `json:"dry_run"`
	MaxRetries         int      `json:"max_retries"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	ParallelUploads    int      `json:"parallel_uploads"`
	MultipartThreshold int64    `json:"multipart_threshold"`
	MaxConcurrency     int      `json:"max_concurrency"`
	MultipartChunksize int64    `json:"multipart_chunksize"`
	UseThreads         bool     `json:"use_threads"`
	MaxIOQueue         int      `json:"max_io_queue"`
	IOChunksize        int      `json:"io_chunksize"`
	EnableProgress     bool     `json:"enable_progress"`
}

// UploadTask 単一のアップロードタスクを表す構造体
type UploadTask struct {
	Name        string  `json:"name"`
	Description *string `json:"description"` // nullableなのでポインタ
	Source      string  `json:"source"`
	Bucket      string  `json:"bucket"`
	S3Key       *string `json:"s3_key"`        // 単一ファイル用
	S3KeyPrefix *string `json:"s3_key_prefix"` // ディレクトリ用
	Recursive   bool    `json:"recursive"`
	Enabled     bool    `json:"enabled"`
}

// Config メイン設定構造体
type Config struct {
	Logging     LoggingConfig `json:"logging"`
	AWS         AWSConfig     `json:"aws"`
	Options     UploadOptions `json:"options"`
	UploadTasks []UploadTask  `json:"upload_tasks"`
}

// LoadFromFile JSONファイルから設定を読み込む
func LoadFromFile(configPath string) (*Config, error) {
	// ファイルが存在するかチェック
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file %s not found", configPath)
	}

	// ファイルを読み込み
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	// JSONをパース
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	// バリデーション
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// Validate 設定全体の検証を行う
func (c *Config) Validate() error {
	// AssumeRole設定の検証
	if c.AWS.AssumeRole != nil {
		if err := c.AWS.AssumeRole.Validate(); err != nil {
			return err
		}
	}

	// 基本的な設定値の検証
	if c.AWS.Region == "" {
		return fmt.Errorf("AWS region cannot be empty")
	}

	if c.Options.ParallelUploads < 1 {
		return fmt.Errorf("parallel_uploads must be at least 1")
	}

	if c.Options.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	return nil
}
