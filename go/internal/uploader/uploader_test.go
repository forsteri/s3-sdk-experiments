package uploader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"s3-uploader/internal/models"
)

// MockS3Operations S3操作のモック
type MockS3Operations struct {
	UploadFileCalled bool
	UploadFileError  error
	Files            map[string][]byte
}

func (m *MockS3Operations) UploadFile(ctx context.Context, bucket, key, filePath string) error {
	m.UploadFileCalled = true
	if m.UploadFileError != nil {
		return m.UploadFileError
	}
	// 成功時の処理
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[key] = []byte("mock content")
	return nil
}

func (m *MockS3Operations) UploadFileWithMetadata(ctx context.Context, bucket, key, filePath string, metadata map[string]string) error {
	return m.UploadFile(ctx, bucket, key, filePath)
}

func (m *MockS3Operations) ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	var objects []types.Object
	now := time.Now()
	
	for k, v := range m.Files {
		// prefixフィルタリング
		if prefix != "" && len(k) < len(prefix) {
			continue
		}
		if prefix != "" && k[:len(prefix)] != prefix {
			continue
		}
		
		size := int64(len(v))
		objects = append(objects, types.Object{
			Key:          &k,
			Size:         &size,
			LastModified: &now,
		})
	}
	return objects, nil
}

func (m *MockS3Operations) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, exists := m.Files[key]
	return exists, nil
}

func (m *MockS3Operations) UploadFileMultipart(ctx context.Context, bucket, key, filePath string, chunkSize int64, metadata map[string]string) error {
	return m.UploadFile(ctx, bucket, key, filePath)
}

func (m *MockS3Operations) UploadFileMultipartParallel(ctx context.Context, bucket, key, filePath string, chunkSize int64, numWorkers int, metadata map[string]string) error {
	return m.UploadFile(ctx, bucket, key, filePath)
}

func TestUploadFile_DryRun(t *testing.T) {
	// モックのS3クライアントを作成
	mockS3 := &MockS3Operations{}
	
	// テスト用の設定
	config := models.UploadOptions{
		DryRun:         true,
		MaxRetries:     3,
		ExcludePatterns: []string{"*.tmp"},
	}
	
	// Uploaderを作成
	uploader := NewUploader(mockS3, config)
	
	// テスト実行
	ctx := context.Background()
	result, err := uploader.UploadFile(ctx, "../test-data/sample_data.csv", "test-bucket", "test-key")
	
	// 検証
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if !result.Success {
		t.Error("Expected success to be true")
	}
	
	if result.SkippedReason != "dry run mode" {
		t.Errorf("Expected skip reason 'dry run mode', got '%s'", result.SkippedReason)
	}
	
	// ドライランモードではS3操作が呼ばれないことを確認
	if mockS3.UploadFileCalled {
		t.Error("S3 upload should not be called in dry run mode")
	}
}

func TestUploadFile_WithError(t *testing.T) {
	// エラーを返すモック
	mockS3 := &MockS3Operations{
		UploadFileError: errors.New("network error"),
	}
	
	config := models.UploadOptions{
		DryRun:     false,
		MaxRetries: 0, // リトライなし
	}
	
	uploader := NewUploader(mockS3, config)
	
	ctx := context.Background()
	result, err := uploader.UploadFile(ctx, "../test-data/sample_data.csv", "test-bucket", "test-key")
	
	// エラーが返されることを確認
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	if result.Success {
		t.Error("Expected success to be false")
	}
}
