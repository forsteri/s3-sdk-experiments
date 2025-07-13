#!/bin/bash

# デバッグ用のヘルパースクリプト

# 1. まず、バケットのリージョンを確認
echo "=== バケットのリージョン確認 ==="
BUCKET="${1:-s3-experiment-bucket-250615}"
echo "Checking bucket: $BUCKET"

# バケットのリージョンを取得
REGION=$(aws s3api get-bucket-location --bucket "$BUCKET" 2>/dev/null | jq -r '.LocationConstraint // "us-east-1"')
echo "Bucket region: $REGION"

# 2. 正しいリージョンを使ってテスト
echo -e "\n=== 正しいリージョンでマルチパートアップロードをテスト ==="
if [ -f "../test-data/test-150mb.bin" ]; then
    echo "Testing with region: $REGION"
    
    # config.jsonを一時的に修正（バックアップを作成）
    cp config.json config.json.bak
    
    # jqを使ってリージョンを更新
    jq --arg region "$REGION" '.aws.region = $region' config.json > config.json.tmp && mv config.json.tmp config.json
    
    # テストを実行
    echo "Running multipart upload test..."
    go run cmd/multipart-test/main.go \
        -source ../test-data/test-150mb.bin \
        -bucket "$BUCKET" \
        -key test/large-$(date +%Y%m%d-%H%M%S).bin
    
    # 設定を元に戻す
    mv config.json.bak config.json
else
    echo "Test file not found: ../test-data/test-150mb.bin"
    echo "Creating a test file..."
    mkdir -p ../test-data
    dd if=/dev/urandom of=../test-data/test-150mb.bin bs=1M count=150
fi
