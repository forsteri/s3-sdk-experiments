#!/bin/bash
set -euo pipefail
# S3 Uploader 統合テストスクリプト

echo "=== S3 Uploader Integration Test ==="
echo ""

# カラー定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# テスト結果を記録
PASSED=0
FAILED=0

# テスト関数
run_test() {
    local test_name=$1
    local command=$2
    
    echo -n "Testing: $test_name ... "
    
    if eval "$command" > /tmp/test_output.log 2>&1; then
        echo -e "${GREEN}PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAILED${NC}"
        echo "Error output:"
        tail -20 /tmp/test_output.log
        ((FAILED++))
    fi
}

# 1. 依存関係のチェック
echo "1. Checking dependencies..."
run_test "Go installation" "go version"
run_test "AWS credentials" "test -f ~/.aws/credentials || test -n '$AWS_PROFILE'"

# 2. ビルドテスト
echo -e "\n2. Build tests..."
run_test "Main build" "go build -o bin/s3-uploader main.go"
run_test "Task runner build" "go build -o bin/task-runner cmd/task-runner/main.go"

# 3. ユニットテスト
echo -e "\n3. Unit tests..."
run_test "Fileutils tests" "go test ./internal/fileutils/"
run_test "Fileutils coverage" "go test -cover ./internal/fileutils/ | grep -q coverage"

# 4. テストデータの準備
echo -e "\n4. Preparing test data..."
mkdir -p test-integration-data
echo "test content" > test-integration-data/test1.txt
echo "another test" > test-integration-data/test2.txt
mkdir -p test-integration-data/subdir
echo "nested content" > test-integration-data/subdir/nested.txt

# 5. 基本機能テスト
echo -e "\n5. Basic functionality tests..."
run_test "Dry run mode" "./bin/s3-uploader -dry-run -test"
run_test "File scan test" "go run cmd/scan-test/main.go"

# 6. アップロードテスト（ドライラン）
echo -e "\n6. Upload tests (dry-run)..."
run_test "Single file upload (dry)" "go run cmd/upload-test/main.go -source test-integration-data/test1.txt -key test/test1.txt -dry-run"
run_test "Directory upload (dry)" "go run cmd/upload-test/main.go -source test-integration-data -key test-dir/ -recursive -dry-run"

# 7. 並列処理テスト
echo -e "\n7. Parallel processing tests..."
run_test "Parallel with 1 worker" "go run cmd/parallel-test/main.go -source test-integration-data -recursive -workers 1 -dry-run"
run_test "Parallel with 4 workers" "go run cmd/parallel-test/main.go -source test-integration-data -recursive -workers 4 -dry-run"

# 8. エラーハンドリングテスト
echo -e "\n8. Error handling tests..."
run_test "Non-existent file" "! go run cmd/upload-test/main.go -source non-existent.txt -key test/error.txt -dry-run"
run_test "Invalid config" "! ./bin/s3-uploader -config non-existent.json"

# 9. リトライ機能テスト
echo -e "\n9. Retry functionality tests..."
# config.jsonを一時的に変更してmax_retriesを確認
cp config.json config.json.test-backup
jq '.options.max_retries = 2' config.json > config.json.tmp && mv config.json.tmp config.json
run_test "Retry configuration" "grep -q 'max_retries.*2' config.json"
mv config.json.test-backup config.json

# 10. クリーンアップ
echo -e "\n10. Cleanup..."
rm -rf test-integration-data
rm -f bin/s3-uploader bin/task-runner
rm -f /tmp/test_output.log

# 結果サマリー
echo -e "\n=== Test Summary ==="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi
