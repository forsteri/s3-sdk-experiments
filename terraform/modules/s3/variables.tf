variable "bucket_name" {
  description = "S3バケット名"
  type        = string
}

variable "force_destroy" {
  description = "バケット内にオブジェクトがあっても強制削除するか"
  type        = bool
  default     = false
}

variable "enable_versioning" {
  description = "バージョニングを有効にするか"
  type        = bool
  default     = true
}

variable "enable_encryption" {
  description = "暗号化を有効にするか"
  type        = bool
  default     = true
}

variable "kms_key_id" {
  description = "KMSキーID（nullの場合はAES256を使用）"
  type        = string
  default     = null
}

variable "block_public_access" {
  description = "パブリックアクセスをブロックするか"
  type        = bool
  default     = true
}

variable "lifecycle_rules" {
  description = "ライフサイクルルール"
  type = list(object({
    id              = string
    enabled         = bool
    expiration_days = optional(number)
    transitions = list(object({
      days          = number
      storage_class = string
    }))
  }))
  default = []
}

variable "tags" {
  description = "タグ"
  type        = map(string)
  default     = {}
}