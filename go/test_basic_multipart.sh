#!/bin/bash
set -euo pipefail
# マルチパートアップロードの基本テストスクリプト

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

# テスト用のバケット名（環境変数から取得するか、デフォルト値を使用）
BUCKET="${S3_TEST_BUCKET:-datalake-poc-raw-891376985958}"

echo -e "${BLUE}=== S3 Multipart Upload Test ===${NC}"
echo "Project directory: $PROJECT_DIR"
echo "Test data directory: $TEST_DATA_DIR"
echo "Target bucket: $BUCKET"
echo ""

# 1. 通常のアップロードテスト（小さいファイル）
echo -e "${GREEN}1. Testing normal upload with small file...${NC}"
echo "File: sample_data.csv (72 bytes)"
echo ""

cd "$GO_DIR" || exit 1
go run cmd/upload-test/main.go \
    -source "$TEST_DATA_DIR/sample_data.csv" \
    -key "test/normal-upload/sample.csv" \
    -dry-run

echo ""
read -p "Continue with actual upload? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    go run cmd/upload-test/main.go \
        -source "$TEST_DATA_DIR/sample_data.csv" \
        -key "test/normal-upload/sample.csv"
fi

echo ""
echo -e "${GREEN}2. Testing forced multipart upload with medium file...${NC}"

# medium ディレクトリから最初のファイルを選択
MEDIUM_FILE=$(find "$TEST_DATA_DIR/benchmark/medium" -type f -name "*.bin" | head -1)
FILE_SIZE=$(ls -lh "$MEDIUM_FILE" | awk '{print $5}')
FILE_NAME=$(basename "$MEDIUM_FILE")

echo "File: $FILE_NAME ($FILE_SIZE)"
echo ""

# 強制的にマルチパートアップロードを使用（チャンクサイズ 5MB）
cd "$GO_DIR" || exit 1
go run cmd/multipart-test/main.go \
    -source "$MEDIUM_FILE" \
    -bucket "$BUCKET" \
    -key "test/multipart/$FILE_NAME" \
    -multipart \
    -chunk-size 5 \
    -dry-run

echo ""
read -p "Continue with actual multipart upload? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    go run cmd/multipart-test/main.go \
        -source "$MEDIUM_FILE" \
        -bucket "$BUCKET" \
        -key "test/multipart/$FILE_NAME" \
        -multipart \
        -chunk-size 5
fi

echo ""
echo -e "${GREEN}3. Testing parallel multipart upload with large directory...${NC}"
echo "Directory: benchmark/large (89MB total)"
echo ""

# largeディレクトリから大きめのファイルを選択
LARGE_FILE=$(find "$TEST_DATA_DIR/benchmark/large" -type f -name "*.bin" -size +5M | head -1)
if [ -n "$LARGE_FILE" ]; then
    FILE_SIZE=$(ls -lh "$LARGE_FILE" | awk '{print $5}')
    FILE_NAME=$(basename "$LARGE_FILE")
    
    echo "Selected file: $FILE_NAME ($FILE_SIZE)"
    echo "Testing with different worker counts..."
    echo ""
    
    # 2ワーカーでテスト
    echo -e "${YELLOW}Testing with 2 workers...${NC}"
    time go run cmd/multipart-test/main.go \
        -source "$LARGE_FILE" \
        -bucket "$BUCKET" \
        -key "test/parallel-2/$FILE_NAME" \
        -workers 2 \
        -chunk-size 5
    
    echo ""
    
    # 4ワーカーでテスト
    echo -e "${YELLOW}Testing with 4 workers...${NC}"
    time go run cmd/multipart-test/main.go \
        -source "$LARGE_FILE" \
        -bucket "$BUCKET" \
        -key "test/parallel-4/$FILE_NAME" \
        -workers 4 \
        -chunk-size 5
fi

echo ""
echo -e "${GREEN}4. Testing automatic multipart selection...${NC}"
echo "This should automatically use multipart for files > 100MB"
echo ""

# 大きなテストファイルを作成（もしなければ）
LARGE_TEST_FILE="$TEST_DATA_DIR/test-150mb.bin"
if [ ! -f "$LARGE_TEST_FILE" ]; then
    echo "Creating 150MB test file..."
    dd if=/dev/zero of="$LARGE_TEST_FILE" bs=1M count=150 2>/dev/null
fi

echo "File: test-150mb.bin (150MB)"
echo ""

cd "$GO_DIR" || exit 1
go run cmd/multipart-test/main.go \
    -source "$LARGE_TEST_FILE" \
    -bucket "$BUCKET" \
    -key "test/auto-multipart/test-150mb.bin" \
    -dry-run

echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo "Check the logs for:"
echo "1. 'Using standard upload' for small files"
echo "2. 'Multipart upload created' for multipart uploads"
echo "3. 'Worker uploaded part' for parallel uploads"
echo "4. Transfer speeds and completion times"
echo ""

# S3の結果を確認
echo -e "${GREEN}Uploaded files in S3:${NC}"
aws s3 ls "s3://$BUCKET/test/" --recursive --human-readable 2>/dev/null || echo "AWS CLI not configured or bucket not accessible"
