#!/bin/bash

# テスト用ヘルパースクリプト

# カラー定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# デフォルト値
BUCKET="${S3_TEST_BUCKET:-your-test-bucket}"
TEST_DIR="./test-files"

# テストディレクトリを作成
mkdir -p "$TEST_DIR"

# 使い方を表示
usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  create-files    - テスト用ファイルを作成"
    echo "  basic-test      - 基本的なマルチパートアップロードテスト"
    echo "  benchmark       - 性能ベンチマーク"
    echo "  cleanup         - テストファイルをクリーンアップ"
    echo "  list-incomplete - 不完全なマルチパートアップロードを表示"
    echo ""
    echo "Environment variables:"
    echo "  S3_TEST_BUCKET  - テスト用バケット名 (default: $BUCKET)"
}

# テストファイルを作成
create_test_files() {
    echo -e "${GREEN}Creating test files...${NC}"
    
    # 小さいファイル（10MB）
    if [ ! -f "$TEST_DIR/test-10mb.bin" ]; then
        echo "Creating 10MB test file..."
        dd if=/dev/zero of="$TEST_DIR/test-10mb.bin" bs=1M count=10 2>/dev/null
    fi
    
    # 中サイズ（150MB）
    if [ ! -f "$TEST_DIR/test-150mb.bin" ]; then
        echo "Creating 150MB test file..."
        dd if=/dev/zero of="$TEST_DIR/test-150mb.bin" bs=1M count=150 2>/dev/null
    fi
    
    # 大サイズ（500MB）
    if [ ! -f "$TEST_DIR/test-500mb.bin" ]; then
        echo "Creating 500MB test file..."
        dd if=/dev/zero of="$TEST_DIR/test-500mb.bin" bs=1M count=500 2>/dev/null
    fi
    
    echo -e "${GREEN}Test files created!${NC}"
    ls -lh "$TEST_DIR"
}

# 基本テスト
basic_test() {
    echo -e "${GREEN}Running basic multipart upload test...${NC}"
    
    # ドライラン
    echo -e "${YELLOW}1. Dry run test...${NC}"
    go run cmd/multipart-test/main.go \
        -source "$TEST_DIR/test-10mb.bin" \
        -bucket "$BUCKET" \
        -key "test/basic-multipart.bin" \
        -multipart \
        -chunk-size 2 \
        -dry-run
    
    echo ""
    read -p "Continue with actual upload? (y/n) " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}2. Actual upload test...${NC}"
        go run cmd/multipart-test/main.go \
            -source "$TEST_DIR/test-10mb.bin" \
            -bucket "$BUCKET" \
            -key "test/basic-multipart.bin" \
            -multipart \
            -chunk-size 2
    fi
}

# ベンチマーク
benchmark() {
    echo -e "${GREEN}Running benchmark tests...${NC}"
    
    if [ ! -f "$TEST_DIR/test-500mb.bin" ]; then
        echo "Creating 500MB test file for benchmark..."
        dd if=/dev/zero of="$TEST_DIR/test-500mb.bin" bs=1M count=500 2>/dev/null
    fi
    
    go run cmd/multipart-test/main.go \
        -source "$TEST_DIR/test-500mb.bin" \
        -bucket "$BUCKET" \
        -key "test/benchmark" \
        -benchmark \
        -workers 8
}

# クリーンアップ
cleanup() {
    echo -e "${YELLOW}Cleaning up test files...${NC}"
    
    read -p "Delete local test files? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$TEST_DIR"
        echo -e "${GREEN}Local files deleted${NC}"
    fi
    
    read -p "Delete S3 test objects? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        aws s3 rm "s3://$BUCKET/test/" --recursive
        echo -e "${GREEN}S3 objects deleted${NC}"
    fi
}

# 不完全なマルチパートアップロードを表示
list_incomplete() {
    echo -e "${GREEN}Listing incomplete multipart uploads...${NC}"
    aws s3api list-multipart-uploads --bucket "$BUCKET" --query 'Uploads[*].[Key,UploadId,Initiated]' --output table
}

# メインロジック
case "$1" in
    create-files)
        create_test_files
        ;;
    basic-test)
        create_test_files
        basic_test
        ;;
    benchmark)
        create_test_files
        benchmark
        ;;
    cleanup)
        cleanup
        ;;
    list-incomplete)
        list_incomplete
        ;;
    *)
        usage
        exit 1
        ;;
esac
