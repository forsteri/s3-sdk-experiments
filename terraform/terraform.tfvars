# AWS設定
aws_region  = "ap-northeast-1"
aws_profile = "sandbox01"
environment = "dev"

# S3設定
experiment_bucket_name = "s3-experiment-bucket-250615"
enable_versioning     = true
enable_monitoring     = false

# 追加タグ
additional_tags = {
  Owner       = "forsteri"
  Project     = "S3-SDK-Experiments"
}