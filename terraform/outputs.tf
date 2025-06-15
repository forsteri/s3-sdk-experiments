# バケット情報の出力
output "experiment_bucket_name" {
  description = "実験用S3バケット名"
  value       = module.experiment_bucket.bucket_id
}

output "experiment_bucket_arn" {
  description = "実験用S3バケットARN"
  value       = module.experiment_bucket.bucket_arn
}

output "experiment_bucket_domain_name" {
  description = "S3バケットのドメイン名"
  value       = module.experiment_bucket.bucket_domain_name
}

output "experiment_bucket_region" {
  description = "S3バケットのリージョン"
  value       = module.experiment_bucket.bucket_region
}

# 設定サマリー
output "configuration_summary" {
  description = "設定サマリー"
  value = {
    bucket_name = module.experiment_bucket.bucket_id
    region      = var.aws_region
    environment = var.environment
    versioning  = var.enable_versioning
    monitoring  = var.enable_monitoring
  }
}
