package main

import (
	"fmt"
	"log"
	"os"

	"s3-uploader/internal/fileutils"
	"s3-uploader/internal/models"
)

func main() {
	fmt.Println("📁 ファイルスキャン機能のテスト")

	// 設定を読み込み
	cfg, err := models.LoadFromFile("config.json")
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// FileScannerを作成
	scanner := fileutils.NewFileScanner(cfg.Options.ExcludePatterns)

	// 各アップロードタスクをテスト
	for _, task := range cfg.UploadTasks {
		if !task.Enabled {
			fmt.Printf("\n⏭️  タスク '%s' はスキップ（無効）\n", task.Name)
			continue
		}

		fmt.Printf("\n🔍 タスク: %s\n", task.Name)
		fmt.Printf("   説明: %s\n", getDescription(task.Description))
		fmt.Printf("   ソース: %s\n", task.Source)

		// ファイルかディレクトリかを判定
		info, err := os.Stat(task.Source)
		if err != nil {
			fmt.Printf("   ❌ エラー: %v\n", err)
			continue
		}

		if info.IsDir() {
			// ディレクトリの場合
			fmt.Printf("   タイプ: ディレクトリ（再帰: %v）\n", task.Recursive)
			
			files, err := scanner.ScanDirectory(task.Source, task.Recursive)
			if err != nil {
				fmt.Printf("   ❌ スキャンエラー: %v\n", err)
				continue
			}

			fmt.Printf("   ✅ %d個のファイルが見つかりました:\n", len(files))
			
			// 最初の5ファイルを表示
			for i, file := range files {
				if i >= 5 {
					fmt.Printf("      ... 他 %d ファイル\n", len(files)-5)
					break
				}
				fmt.Printf("      - %s (%s)\n", file.RelativePath, formatSize(file.Size))
			}

			// 合計サイズを計算
			var totalSize int64
			for _, file := range files {
				totalSize += file.Size
			}
			fmt.Printf("   📊 合計サイズ: %s\n", formatSize(totalSize))

		} else {
			// 単一ファイルの場合
			fmt.Println("   タイプ: 単一ファイル")
			
			fileInfo, err := scanner.GetFileInfo(task.Source)
			if err != nil {
				fmt.Printf("   ❌ ファイル情報取得エラー: %v\n", err)
				continue
			}

			fmt.Printf("   ✅ ファイル情報:\n")
			fmt.Printf("      - 名前: %s\n", fileInfo.Name())
			fmt.Printf("      - サイズ: %s\n", formatSize(fileInfo.Size))
			fmt.Printf("      - パス: %s\n", fileInfo.Path)
		}
	}

	fmt.Println("\n✨ スキャン完了!")
}

// 説明を取得（nilチェック付き）
func getDescription(desc *string) string {
	if desc == nil || *desc == "" {
		return "(説明なし)"
	}
	return *desc
}

// ファイルサイズを人間が読みやすい形式に変換
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
