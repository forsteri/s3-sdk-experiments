package fileutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo ファイル情報を保持する構造体
type FileInfo struct {
	Path         string // 絶対パス
	Size         int64  // ファイルサイズ（バイト）
	RelativePath string // 基準ディレクトリからの相対パス
}

// Name ファイル名を返す
func (f *FileInfo) Name() string {
	return filepath.Base(f.Path)
}

// FileScanner ファイルスキャン機能を提供
type FileScanner struct {
	excludePatterns []string
}

// NewFileScanner 新しいFileScannerを作成
func NewFileScanner(excludePatterns []string) *FileScanner {
	return &FileScanner{
		excludePatterns: excludePatterns,
	}
}

// ShouldExclude ファイルが除外パターンに一致するかチェック
func (fs *FileScanner) ShouldExclude(filePath string) bool {
	fileName := filepath.Base(filePath)

	for _, pattern := range fs.excludePatterns {
		// ファイル名でのマッチ
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		// パス全体でのマッチ（パターンが含まれているかチェック）
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

// ScanDirectory ディレクトリをスキャンしてファイル情報を取得
func (fs *FileScanner) ScanDirectory(directory string, recursive bool) ([]FileInfo, error) {
	// ディレクトリの存在確認
	info, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", directory)
	}

	// 絶対パスに変換
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	var fileInfos []FileInfo

	if recursive {
		// 再帰的スキャン
		err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// エラーが発生してもスキャンを続行
				// TODO: ログに記録する
				return nil
			}

			// ディレクトリ自体はスキップ
			if d.IsDir() {
				// 除外パターンに一致するディレクトリは配下もスキップ
				if fs.ShouldExclude(path) && path != absDir {
					return filepath.SkipDir
				}
				return nil
			}

			// 除外パターンチェック
			if fs.ShouldExclude(path) {
				return nil
			}

			// ファイル情報を取得
			info, err := d.Info()
			if err != nil {
				// エラーが発生してもスキャンを続行
				return nil
			}

			// 相対パスを計算
			relPath, err := filepath.Rel(absDir, path)
			if err != nil {
				return nil
			}

			fileInfos = append(fileInfos, FileInfo{
				Path:         path,
				Size:         info.Size(),
				RelativePath: relPath,
			})

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("directory scan failed: %w", err)
		}
	} else {
		// 非再帰的スキャン（直下のファイルのみ）
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			// ディレクトリはスキップ
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(absDir, entry.Name())

			// 除外パターンチェック
			if fs.ShouldExclude(path) {
				continue
			}

			// ファイル情報を取得
			info, err := entry.Info()
			if err != nil {
				// エラーが発生してもスキャンを続行
				continue
			}

			fileInfos = append(fileInfos, FileInfo{
				Path:         path,
				Size:         info.Size(),
				RelativePath: entry.Name(),
			})
		}
	}

	return fileInfos, nil
}

// GetFileInfo 単一ファイルの情報を取得
func (fs *FileScanner) GetFileInfo(filePath string) (*FileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("not a file: %s", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &FileInfo{
		Path:         absPath,
		Size:         info.Size(),
		RelativePath: filepath.Base(absPath),
	}, nil
}
