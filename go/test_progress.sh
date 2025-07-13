#!/bin/bash

# 進捗表示機能のテストスクリプト

echo "🚀 S3 Uploader Progress Test"
echo "============================"

# 1. 単一ファイルのアップロード（進捗表示なし）
echo -e "\n1. Testing single file upload..."
go run cmd/upload-test/main.go -source ../test-data/sample_data.csv -key test/sample.csv

# 2. ディレクトリの並列アップロード（進捗表示あり）
echo -e "\n2. Testing directory parallel upload with progress..."
go run cmd/parallel-test/main.go -source ../test-data -recursive

# 3. ベンチマークモード（順次 vs 並列の比較）
echo -e "\n3. Running benchmark (sequential vs parallel)..."
go run cmd/parallel-test/main.go -benchmark ../test-data

echo -e "\n✅ All tests completed!"

