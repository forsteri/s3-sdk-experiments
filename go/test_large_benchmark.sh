#!/bin/bash

# 大容量ファイルでのベンチマークテスト

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_DIR="/Users/forsteri/work/Projects/s3-sdk-experiments"
GO_DIR="$PROJECT_DIR/go"
TEST_DATA_DIR="$PROJECT_DIR/test-data"
BUCKET="s3-experiment-bucket-250615"

echo -e "${BLUE}=== Large File Benchmark Test ===${NC}"
echo ""

cd "$GO_DIR"

# 1. さらに大きなファイルを作成（500MB、1GB）
echo -e "${GREEN}Creating test files...${NC}"

if [ ! -f "$TEST_DATA_DIR/test-500mb.bin" ]; then
    echo "Creating 500MB test file..."
    dd if=/dev/zero of="$TEST_DATA_DIR/test-500mb.bin" bs=1M count=500 2>/dev/null
fi

if [ ! -f "$TEST_DATA_DIR/test-1gb.bin" ]; then
    echo "Creating 1GB test file..."
    dd if=/dev/zero of="$TEST_DATA_DIR/test-1gb.bin" bs=1M count=1024 2>/dev/null
fi

echo ""
echo -e "${GREEN}Testing different configurations on large files${NC}"
echo ""

# 500MBファイルでのテスト
echo -e "${YELLOW}=== 500MB File Test ===${NC}"
for workers in 2 4 8 16; do
    for chunk in 5 10 20; do
        echo -e "${BLUE}Workers: $workers, Chunk: ${chunk}MB${NC}"
        
        time go run cmd/multipart-test/main.go \
            -source "$TEST_DATA_DIR/test-500mb.bin" \
            -bucket "$BUCKET" \
            -key "benchmark/500mb-w${workers}-c${chunk}.bin" \
            -workers $workers \
            -chunk-size $chunk 2>&1 | grep -E "(throughput|duration|total_parts)"
        
        echo ""
    done
done

# 1GBファイルでのテスト（オプション）
echo ""
read -p "Test with 1GB file? This may take a while (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}=== 1GB File Test ===${NC}"
    
    # 最適と思われる設定でテスト
    go run cmd/multipart-test/main.go \
        -source "$TEST_DATA_DIR/test-1gb.bin" \
        -bucket "$BUCKET" \
        -key "benchmark/1gb-optimal.bin" \
        -workers 8 \
        -chunk-size 20
fi

echo ""
echo -e "${GREEN}=== Performance Tips ===${NC}"
echo "1. Monitor network bandwidth usage during upload"
echo "2. Optimal chunk size depends on network stability"
echo "3. More workers isn't always better (diminishing returns)"
echo "4. Consider memory usage with large chunk sizes"
