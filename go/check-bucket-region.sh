#!/bin/bash

# バケットのリージョンを確認するスクリプト
BUCKET_NAME="${1:-your-bucket}"

echo "Checking region for bucket: $BUCKET_NAME"

# AWS CLIでバケットのリージョンを確認
aws s3api get-bucket-location --bucket "$BUCKET_NAME" 2>/dev/null | jq -r '.LocationConstraint // "us-east-1"'

# または、バケットの情報を表示
echo -e "\nBucket details:"
aws s3api head-bucket --bucket "$BUCKET_NAME" 2>&1 | grep -E "x-amz-bucket-region|BucketRegion"
