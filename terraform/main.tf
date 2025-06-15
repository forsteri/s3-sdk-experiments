terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile

  default_tags {
    tags = {
      Project     = "S3-SDK-Experiments"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# 実験用バケット
module "experiment_bucket" {
  source = "./modules/s3"

  bucket_name         = var.experiment_bucket_name
  force_destroy       = true
  enable_versioning   = var.enable_versioning
  enable_encryption   = true
  block_public_access = true

  lifecycle_rules = [
    {
      id      = "experiment_data_lifecycle"
      enabled = true
      transitions = [
        {
          days          = 30
          storage_class = "STANDARD_IA"
        },
        {
          days          = 90
          storage_class = "GLACIER"
        }
      ]
      expiration_days = 365
    }
  ]

  tags = merge(var.additional_tags, {
    Name    = "S3-Experiment-Bucket"
    Purpose = "SDK-Experiments-add-Benchmarks"
  })
}

# テスト用バケット（小さなファイル用）
module "test_bucket_small" {
  source = "./modules/s3"

  bucket_name       = "${var.experiment_bucket_name}-small-files"
  force_destroy     = true
  enable_versioning = false
  enable_encryption = true

  lifecycle_rules = [
    {
      id              = "small_files_cleanup"
      enabled         = true
      transitions     = []
      expiration_days = 30 # 30日で削除
    }
  ]

  tags = merge(var.additional_tags, {
    Name    = "Small-Files-Test-Bucket"
    Purpose = "Small-File-Upload-Tests"
  })
}

# 大容量ファイル用バケット
module "test_bucket_large" {
  source = "./modules/s3"
  
  bucket_name        = "${var.experiment_bucket_name}-large-files"
  enable_versioning  = true
  enable_encryption  = true
  
  lifecycle_rules = [
    {
      id      = "large_files_management"
      enabled = true
      transitions = [
        {
          days          = 30  # 30日以上に変更
          storage_class = "STANDARD_IA"
        },
        {
          days          = 90  # 90日でGlacierに移行
          storage_class = "GLACIER"
        }
      ]
      expiration_days = 180
    }
  ]
  
  tags = merge(var.additional_tags, {
    Name    = "Large-Files-Test-Bucket"
    Purpose = "Large-File-Upload-Tests"
  })
}