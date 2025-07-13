#!/bin/bash
set -euo pipefail

# ベンチマークディレクトリを使った包括的なテストスクリプト

# カラー定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# プロジェクトディレクトリ
PROJECT_DIR="/Users/forsteri/work/Projects/s3-sdk-experiments"
GO_DIR="$PROJECT_DIR/go"
TEST_DATA_DIR="$PROJECT_DIR/test-data/benchmark"
RESULTS_FILE="$GO_DIR/benchmark_results.txt"

# テスト用のバケット名
BUCKET="${S3_TEST_BUCKET:-datalake-poc-raw-891376985958}"

# 結果を記録する関数
log_result() {
    echo "$1" | tee -a "$RESULTS_FILE"
}

# ヘッダー表示
echo -e "${MAGENTA}=== S3 Upload Benchmark Test ===${NC}"
echo "Date: $(date)"
echo "Bucket: $BUCKET"
echo ""

# 結果ファイルの初期化
echo "=== S3 Upload Benchmark Results ===" > "$RESULTS_FILE"
echo "Date: $(date)" >> "$RESULTS_FILE"
echo "Bucket: $BUCKET" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# 1. 小さいファイルのディレクトリアップロード比較
echo -e "${CYAN}1. Small Files Directory Upload (100 files, ~7KB each)${NC}"
log_result "1. Small Files Directory Upload"
log_result "Directory: benchmark/small (Total: 704KB)"
echo ""

cd "$GO_DIR" || exit 1

# 順次アップロード
echo -e "${YELLOW}Sequential upload:${NC}"
START_TIME=$(date +%s)
go run cmd/parallel-test/main.go \
    -source "$TEST_DATA_DIR/small" \
    -recursive \
    -parallel=false \
    -key-prefix "benchmark/small-sequential/" 2>&1 | tee -a "$RESULTS_FILE"
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
log_result "Sequential duration: ${DURATION}s"
echo ""

# 並列アップロード（3ワーカー）
echo -e "${YELLOW}Parallel upload (3 workers):${NC}"
START_TIME=$(date +%s)
go run cmd/parallel-test/main.go \
    -source "$TEST_DATA_DIR/small" \
    -recursive \
    -workers 3 \
    -key-prefix "benchmark/small-parallel-3/" 2>&1 | tee -a "$RESULTS_FILE"
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
log_result "Parallel (3 workers) duration: ${DURATION}s"
echo ""

# 2. 中サイズファイルのベンチマーク
echo -e "${CYAN}2. Medium Files Benchmark (50 files, ~500KB each)${NC}"
log_result ""
log_result "2. Medium Files Benchmark"
log_result "Directory: benchmark/medium (Total: 25MB)"
echo ""

# 最大のファイルを選んでマルチパートアップロードテスト
MEDIUM_LARGE=$(find "$TEST_DATA_DIR/medium" -type f -name "*.bin" | xargs ls -S | head -1)
FILE_SIZE=$(ls -lh "$MEDIUM_LARGE" | awk '{print $5}')

echo -e "${YELLOW}Single large file multipart test:${NC}"
echo "File: $(basename "$MEDIUM_LARGE") ($FILE_SIZE)"
echo ""

# ベンチマークモードで実行
go run cmd/multipart-test/main.go \
    -source "$MEDIUM_LARGE" \
    -bucket "$BUCKET" \
    -key "benchmark/medium-multipart" \
    -benchmark \
    -workers 4 2>&1 | tee -a "$RESULTS_FILE"
echo ""

# 3. 大容量ファイルのマルチパートアップロード比較
echo -e "${CYAN}3. Large Files Multipart Benchmark${NC}"
log_result ""
log_result "3. Large Files Multipart Benchmark"
echo ""

# 最大のファイルを見つける
LARGE_FILE=$(find "$TEST_DATA_DIR/large" -type f -name "*.bin" | xargs ls -S | head -1)
FILE_SIZE=$(ls -lh "$LARGE_FILE" | awk '{print $5}')

echo "Selected file: $(basename "$LARGE_FILE") ($FILE_SIZE)"
echo ""

# 異なるワーカー数でベンチマーク
for workers in 1 2 4 8; do
    echo -e "${YELLOW}Testing with $workers worker(s):${NC}"
    START_TIME=$(date +%s)
    
    go run cmd/multipart-test/main.go \
        -source "$LARGE_FILE" \
        -bucket "$BUCKET" \
        -key "benchmark/large-workers-$workers" \
        -workers $workers \
        -chunk-size 5 2>&1 | grep -E "(duration|throughput)" | tee -a "$RESULTS_FILE"
    
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    log_result "Workers: $workers, Duration: ${DURATION}s"
    echo ""
done

# 4. 混合ファイルサイズのディレクトリアップロード
echo -e "${CYAN}4. Mixed Files Directory Upload${NC}"
log_result ""
log_result "4. Mixed Files Directory Upload"
log_result "Directory: benchmark/mixed (Various sizes)"
echo ""

# ディレクトリ全体のベンチマーク
echo -e "${YELLOW}Full directory parallel upload:${NC}"
time go run cmd/parallel-test/main.go \
    -source "$TEST_DATA_DIR/mixed" \
    -recursive \
    -workers 4 \
    -key-prefix "benchmark/mixed/" 2>&1 | tee -a "$RESULTS_FILE"

echo ""
echo -e "${GREEN}=== Benchmark Complete ===${NC}"
echo "Results saved to: $RESULTS_FILE"
echo ""

# 結果のサマリー
echo -e "${BLUE}Summary:${NC}"
echo "1. Check logs for 'throughput_mbps' values"
echo "2. Compare sequential vs parallel upload times"
echo "3. Observe optimal worker count for your network"
echo "4. Monitor CPU and memory usage during tests"
echo ""

# S3の確認
echo -e "${GREEN}Files uploaded to S3:${NC}"
aws s3 ls "s3://$BUCKET/benchmark/" --recursive --summarize --human-readable 2>/dev/null | tail -10
