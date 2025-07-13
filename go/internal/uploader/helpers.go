package uploader

import (
	"path/filepath"
)

// generateS3Key S3キーを生成するヘルパー関数
func generateS3Key(prefix string, relativePath string) string {
	key := filepath.Join(prefix, relativePath)
	// Windowsパスの場合、バックスラッシュをスラッシュに変換
	return filepath.ToSlash(key)
}
