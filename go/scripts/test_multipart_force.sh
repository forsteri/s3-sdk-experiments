#!/bin/bash
set -euo pipefail
# マルチパートアップロードの動作確認用スクリプト

# カラー定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# プロジェクトディレクトリ
PROJECT_DIR="/Users/forsteri/work/Projects/s3-sdk-experiments"
GO_DIR="$PROJECT_DIR/go"
TEST_DATA_DIR="$PROJECT_DIR/test-data"

# バケット名
BUCKET="s3-experiment-bucket-250615"

echo -e "${BLUE}=== Multipart Upload Function Test ===${NC}"
echo ""

cd "$GO_DIR" || { 
    echo -e "${RED}Error: Cannot change to directory $GO_DIR${NC}"
    exit 1
    }

# 1. 強制マルチパートアップロード（小さいファイルでも動作確認）
echo -e "${GREEN}1. Force multipart upload on medium file${NC}"
MEDIUM_FILE="$TEST_DATA_DIR/benchmark/medium/medium_file_3.bin"
FILE_SIZE=$(ls -lh "$MEDIUM_FILE" | awk '{print $5}')
echo "File: $(basename "$MEDIUM_FILE") ($FILE_SIZE)"
echo "Chunk size: 256KB (to create multiple parts)"
echo ""

go run cmd/multipart-test/main.go \
    -source "$MEDIUM_FILE" \
    -bucket "$BUCKET" \
    -key "test/force-multipart/medium_file_3.bin" \
    -multipart \
    -chunk-size 0.25 \
    -workers 2

echo ""
echo -e "${GREEN}2. Testing with larger file (should see more parts)${NC}"
LARGE_FILE="$TEST_DATA_DIR/benchmark/large/large_file_4.bin"
FILE_SIZE=$(ls -lh "$LARGE_FILE" | awk '{print $5}')
echo "File: $(basename "$LARGE_FILE") ($FILE_SIZE)"
echo "Chunk size: 2MB"
echo ""

go run cmd/multipart-test/main.go \
    -source "$LARGE_FILE" \
    -bucket "$BUCKET" \
    -key "test/multipart-parts/large_file_4.bin" \
    -multipart \
    -chunk-size 2 \
    -workers 3

echo ""
echo -e "${GREEN}3. Benchmark mode test${NC}"
echo "This will compare different upload methods"
echo ""

go run cmd/multipart-test/main.go \
    -source "$LARGE_FILE" \
    -bucket "$BUCKET" \
    -key "test/benchmark" \
    -benchmark \
    -workers 4

echo ""
echo -e "${GREEN}4. Testing automatic multipart with 150MB file${NC}"
if [ -f "$TEST_DATA_DIR/test-150mb.bin" ]; then
    echo "Using existing 150MB test file"
    echo ""
    
    go run cmd/multipart-test/main.go \
        -source "$TEST_DATA_DIR/test-150mb.bin" \
        -bucket "$BUCKET" \
        -key "test/auto-multipart/test-150mb.bin" \
        -workers 4 \
        -chunk-size 10
fi

echo ""
echo -e "${BLUE}=== Check the logs for ===${NC}"
echo "1. 'Multipart upload created' with upload_id"
echo "2. 'Part uploaded' messages with part numbers"
echo "3. 'Worker uploaded part' for parallel processing"
echo "4. 'Multipart upload completed' with total_parts count"
echo ""

# S3 CLIでマルチパートアップロードの状態を確認
echo -e "${GREEN}Checking for incomplete multipart uploads:${NC}"
aws s3api list-multipart-uploads \
    --bucket "$BUCKET" \
    --query 'Uploads[*].[Key,UploadId,Initiated]' \
    --output table 2>/dev/null || echo "Unable to list multipart uploads (CLI not configured or no permissions)"
