#!/bin/bash

# ファイルスキャナーのテストを実行
echo "🧪 Running FileScanner tests..."
go test ./internal/fileutils -v

# カバレッジも確認
echo -e "\n📊 Running with coverage..."
go test ./internal/fileutils -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report saved to coverage.html"
