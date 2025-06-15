# AWS基本設定
variable "aws_region" {
  description = "AWSリージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "aws_profile" {
  description = "AWSプロファイル名（Identity Center）"
  type        = string
  default     = "default"
}

variable "environment" {
  description = "環境名"
  type        = string
  default     = "dev"
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "環境は dev, staging, prod のいずれかである必要があります。"
  }
}

# S3設定
variable "experiment_bucket_name" {
  description = "実験用S3バケット名"
  type        = string
  
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]*[a-z0-9]$", var.experiment_bucket_name))
    error_message = "バケット名は小文字、数字、ハイフンのみ使用可能です。"
  }
}

variable "enable_versioning" {
  description = "S3バケットのバージョニングを有効にするか"
  type        = bool
  default     = true
}

variable "enable_monitoring" {
  description = "CloudWatch監視を有効にするか"
  type        = bool
  default     = false
}

# タグ設定
variable "additional_tags" {
  description = "追加のタグ"
  type        = map(string)
  default     = {}
}