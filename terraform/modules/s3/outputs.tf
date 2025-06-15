output "bucket_id" {
  description = "S3バケットID"
  value       = aws_s3_bucket.this.id
}

output "bucket_arn" {
  description = "S3バケットARN"
  value       = aws_s3_bucket.this.arn
}

output "bucket_domain_name" {
  description = "S3バケットドメイン名"
  value       = aws_s3_bucket.this.bucket_domain_name
}

output "bucket_region" {
  description = "S3バケットリージョン"
  value       = aws_s3_bucket.this.region
}