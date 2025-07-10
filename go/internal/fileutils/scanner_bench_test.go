package fileutils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ベンチマーク用の大きなディレクトリ構造を作成
func createBenchmarkDirectory(b *testing.B) (string, func()) {
	tempDir, err := os.MkdirTemp("", "benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	// 100個のファイルと10個のサブディレクトリを作成
	for i := 0; i < 10; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		os.MkdirAll(subdir, 0755)
		
		for j := 0; j < 10; j++ {
			filename := filepath.Join(subdir, fmt.Sprintf("file_%d_%d.txt", i, j))
			content := fmt.Sprintf("This is file %d in subdir %d", j, i)
			os.WriteFile(filename, []byte(content), 0644)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func BenchmarkFileScanner_ScanDirectory_Recursive(b *testing.B) {
	tempDir, cleanup := createBenchmarkDirectory(b)
	defer cleanup()

	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
	}

	scanner := NewFileScanner(excludePatterns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := scanner.ScanDirectory(tempDir, true)
		if err != nil {
			b.Fatalf("ScanDirectory failed: %v", err)
		}
	}
}

func BenchmarkFileScanner_ScanDirectory_NonRecursive(b *testing.B) {
	tempDir, cleanup := createBenchmarkDirectory(b)
	defer cleanup()

	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
	}

	scanner := NewFileScanner(excludePatterns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := scanner.ScanDirectory(tempDir, false)
		if err != nil {
			b.Fatalf("ScanDirectory failed: %v", err)
		}
	}
}

func BenchmarkFileScanner_ShouldExclude(b *testing.B) {
	excludePatterns := []string{
		"*.tmp",
		"*.lock",
		"__pycache__",
		".DS_Store",
		"*.swp",
		"Thumbs.db",
	}

	scanner := NewFileScanner(excludePatterns)
	testPaths := []string{
		"/path/to/file.txt",
		"/path/to/file.tmp",
		"/path/to/.DS_Store",
		"/path/to/__pycache__/module.pyc",
		"/path/to/normal_file.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_ = scanner.ShouldExclude(path)
		}
	}
}
