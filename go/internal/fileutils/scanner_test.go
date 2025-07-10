package fileutils

import (
	"os"
	"path/filepath"
	"testing"
)

// テスト用の一時ディレクトリを作成
func createTestDirectory(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "scanner_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// テストファイルとディレクトリを作成
	files := map[string]string{
		"file1.txt":                     "content1",
		"file2.csv":                     "content2",
		"subdir/file3.txt":              "content3",
		"subdir/file4.tmp":              "temp file",
		"subdir/.DS_Store":              "macos file",
		"subdir/deeper/file5.txt":       "content5",
		"subdir/deeper/__pycache__/file": "cache file",
		"exclude_dir/file6.txt":         "content6",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// クリーンアップ関数
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestFileScanner_ShouldExclude(t *testing.T) {
	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
		"*.swp",
		"Thumbs.db",
	}

	scanner := NewFileScanner(excludePatterns)

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "Normal file",
			filePath: "/path/to/file.txt",
			want:     false,
		},
		{
			name:     "Temp file",
			filePath: "/path/to/file.tmp",
			want:     true,
		},
		{
			name:     "DS_Store file",
			filePath: "/path/to/.DS_Store",
			want:     true,
		},
		{
			name:     "File in __pycache__",
			filePath: "/path/to/__pycache__/module.pyc",
			want:     true,
		},
		{
			name:     "Lock file",
			filePath: "/path/to/database.lock",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanner.ShouldExclude(tt.filePath)
			if got != tt.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestFileScanner_GetFileInfo(t *testing.T) {
	tempDir, cleanup := createTestDirectory(t)
	defer cleanup()

	scanner := NewFileScanner(nil)

	// 正常なファイル
	filePath := filepath.Join(tempDir, "file1.txt")
	info, err := scanner.GetFileInfo(filePath)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Path != filePath {
		absPath, _ := filepath.Abs(filePath)
		if info.Path != absPath {
			t.Errorf("Path = %q, want %q or %q", info.Path, filePath, absPath)
		}
	}

	if info.Size != 8 { // "content1" の長さ
		t.Errorf("Size = %d, want 8", info.Size)
	}

	if info.Name() != "file1.txt" {
		t.Errorf("Name() = %q, want %q", info.Name(), "file1.txt")
	}

	// ディレクトリを指定した場合
	_, err = scanner.GetFileInfo(tempDir)
	if err == nil {
		t.Error("Expected error for directory, got nil")
	}

	// 存在しないファイル
	_, err = scanner.GetFileInfo(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestFileScanner_ScanDirectory_NonRecursive(t *testing.T) {
	tempDir, cleanup := createTestDirectory(t)
	defer cleanup()

	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
		"*.swp",
		"Thumbs.db",
	}

	scanner := NewFileScanner(excludePatterns)

	// 非再帰的スキャン
	files, err := scanner.ScanDirectory(tempDir, false)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// 期待されるファイル: file1.txt, file2.csv（subdirはディレクトリなので含まれない）
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// ファイル名をチェック
	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Name()] = true
	}

	if !fileNames["file1.txt"] || !fileNames["file2.csv"] {
		t.Errorf("Expected files not found. Got: %v", fileNames)
	}
}

func TestFileScanner_ScanDirectory_Recursive(t *testing.T) {
	tempDir, cleanup := createTestDirectory(t)
	defer cleanup()

	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
		"*.swp",
		"Thumbs.db",
	}

	scanner := NewFileScanner(excludePatterns)

	// 再帰的スキャン
	files, err := scanner.ScanDirectory(tempDir, true)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// 期待されるファイル数をチェック
	// file1.txt, file2.csv, subdir/file3.txt, subdir/deeper/file5.txt, exclude_dir/file6.txt
	// 除外: file4.tmp, .DS_Store, __pycache__内のファイル
	expectedCount := 5
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
		for _, f := range files {
			t.Logf("Found: %s", f.RelativePath)
		}
	}

	// 相対パスをチェック
	relativePaths := make(map[string]bool)
	for _, f := range files {
		relativePaths[f.RelativePath] = true
	}

	expectedPaths := []string{
		"file1.txt",
		"file2.csv",
		filepath.Join("subdir", "file3.txt"),
		filepath.Join("subdir", "deeper", "file5.txt"),
		filepath.Join("exclude_dir", "file6.txt"),
	}

	for _, path := range expectedPaths {
		if !relativePaths[path] {
			t.Errorf("Expected file not found: %s", path)
		}
	}

	// 除外されたファイルが含まれていないことを確認
	excludedPaths := []string{
		filepath.Join("subdir", "file4.tmp"),
		filepath.Join("subdir", ".DS_Store"),
	}

	for _, path := range excludedPaths {
		if relativePaths[path] {
			t.Errorf("Excluded file should not be included: %s", path)
		}
	}
}

func TestFileScanner_ScanDirectory_ExcludeDirectory(t *testing.T) {
	tempDir, cleanup := createTestDirectory(t)
	defer cleanup()

	// __pycache__ ディレクトリ全体を除外
	excludePatterns := []string{
		"__pycache__",
	}

	scanner := NewFileScanner(excludePatterns)

	files, err := scanner.ScanDirectory(tempDir, true)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// __pycache__ 内のファイルが含まれていないことを確認
	for _, f := range files {
		if filepath.Base(filepath.Dir(f.Path)) == "__pycache__" {
			t.Errorf("File in __pycache__ should be excluded: %s", f.Path)
		}
	}
}

func TestFileScanner_ScanDirectory_Errors(t *testing.T) {
	scanner := NewFileScanner(nil)

	// 存在しないディレクトリ
	_, err := scanner.ScanDirectory("/nonexistent/directory", false)
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}

	// ファイルを指定した場合
	tempFile, err := os.CreateTemp("", "test_file_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	_, err = scanner.ScanDirectory(tempFile.Name(), false)
	if err == nil {
		t.Error("Expected error for file instead of directory, got nil")
	}
}
