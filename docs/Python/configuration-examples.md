# 設定例集

## 概要

S3 Uploaderの様々な使用シナリオに対応した設定例を紹介します。これらの例を参考に、あなたの用途に合わせた設定を作成してください。

## 基本設定

### 1. 最小限の設定

```json
{
  "logging": {
    "level": "INFO"
  },
  "aws": {
    "region": "ap-northeast-1"
  },
  "options": {},
  "upload_tasks": [
    {
      "name": "単一ファイルアップロード",
      "source": "example.txt",
      "bucket": "my-bucket",
      "s3_key": "example.txt"
    }
  ]
}
```

### 2. 標準設定

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    "file": "logs/s3_uploader.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "default"
  },
  "options": {
    "parallel_uploads": 2,
    "max_retries": 3,
    "enable_progress": true,
    "dry_run": false
  },
  "upload_tasks": [
    {
      "name": "ドキュメントのアップロード",
      "source": "documents/",
      "bucket": "my-document-bucket",
      "s3_key_prefix": "documents/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

## AWS認証設定

### 1. AWSプロファイル使用

```json
{
  "aws": {
    "region": "ap-northeast-1",
    "profile": "production"
  }
}
```

### 2. AssumeRole使用

```json
{
  "aws": {
    "region": "ap-northeast-1",
    "profile": "base-profile",
    "assume_role": {
      "role_arn": "arn:aws:iam::123456789012:role/S3UploaderRole",
      "session_name": "s3-uploader-session",
      "duration_seconds": 3600
    }
  }
}
```

### 3. 外部IDを使用したAssumeRole

```json
{
  "aws": {
    "region": "ap-northeast-1",
    "assume_role": {
      "role_arn": "arn:aws:iam::123456789012:role/CrossAccountRole",
      "session_name": "cross-account-session",
      "external_id": "unique-external-id-12345",
      "duration_seconds": 7200
    }
  }
}
```

## ログ設定

### 1. デバッグログ

```json
{
  "logging": {
    "level": "DEBUG",
    "format": "%(asctime)s - %(name)s - %(levelname)s - %(funcName)s:%(lineno)d - %(message)s",
    "file": "logs/debug.log"
  }
}
```

### 2. 本番環境用ログ

```json
{
  "logging": {
    "level": "WARNING",
    "format": "%(asctime)s - %(levelname)s - %(message)s",
    "file": "logs/production.log"
  }
}
```

### 3. コンソールのみ

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(message)s"
  }
}
```

## アップロードオプション

### 1. 高速アップロード設定

```json
{
  "options": {
    "multipart_threshold": 50000000,
    "max_concurrency": 8,
    "multipart_chunksize": 8388608,
    "parallel_uploads": 4,
    "max_retries": 5,
    "enable_progress": true
  }
}
```

### 2. 安全重視設定

```json
{
  "options": {
    "multipart_threshold": 1000000000,
    "max_concurrency": 2,
    "multipart_chunksize": 5242880,
    "parallel_uploads": 1,
    "max_retries": 10,
    "timeout_seconds": 600,
    "enable_progress": false
  }
}
```

### 3. 帯域制限環境用設定

```json
{
  "options": {
    "multipart_threshold": 200000000,
    "max_concurrency": 2,
    "multipart_chunksize": 5242880,
    "parallel_uploads": 1,
    "max_retries": 5,
    "timeout_seconds": 900,
    "enable_progress": true
  }
}
```

## アップロードタスクの例

### 1. 単一ファイルアップロード

```json
{
  "upload_tasks": [
    {
      "name": "重要なレポート",
      "description": "月次レポートのアップロード",
      "source": "/path/to/monthly-report.pdf",
      "bucket": "company-reports",
      "s3_key": "reports/2024/monthly-report.pdf",
      "enabled": true
    }
  ]
}
```

### 2. ディレクトリアップロード

```json
{
  "upload_tasks": [
    {
      "name": "ログファイルのバックアップ",
      "description": "アプリケーションログの定期バックアップ",
      "source": "/var/log/myapp/",
      "bucket": "log-backup-bucket",
      "s3_key_prefix": "logs/myapp/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 3. 複数タスクの組み合わせ

```json
{
  "upload_tasks": [
    {
      "name": "設定ファイル",
      "source": "config/app.conf",
      "bucket": "config-backup",
      "s3_key": "backups/config/app.conf",
      "enabled": true
    },
    {
      "name": "データベースダンプ",
      "source": "backup/db_dump.sql",
      "bucket": "db-backup",
      "s3_key": "dumps/db_dump.sql",
      "enabled": true
    },
    {
      "name": "ユーザーファイル",
      "source": "uploads/",
      "bucket": "user-files",
      "s3_key_prefix": "user-uploads/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

## 特定用途の設定

### 1. 大容量ファイル用設定

```json
{
  "logging": {
    "level": "INFO",
    "file": "logs/large-files.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "default"
  },
  "options": {
    "multipart_threshold": 500000000,
    "max_concurrency": 10,
    "multipart_chunksize": 104857600,
    "parallel_uploads": 2,
    "max_retries": 3,
    "timeout_seconds": 1800,
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "大容量データファイル",
      "source": "data/large-dataset.zip",
      "bucket": "large-files-bucket",
      "s3_key": "datasets/large-dataset.zip",
      "enabled": true
    }
  ]
}
```

### 2. 画像ファイル用設定

```json
{
  "logging": {
    "level": "INFO",
    "file": "logs/image-upload.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "default"
  },
  "options": {
    "multipart_threshold": 20000000,
    "max_concurrency": 4,
    "multipart_chunksize": 5242880,
    "parallel_uploads": 3,
    "max_retries": 2,
    "exclude_patterns": ["*.tmp", "*.DS_Store", "Thumbs.db"],
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "写真ギャラリー",
      "source": "photos/",
      "bucket": "photo-gallery",
      "s3_key_prefix": "gallery/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 3. ドキュメント管理用設定

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(levelname)s - %(message)s",
    "file": "logs/document-sync.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "documents"
  },
  "options": {
    "multipart_threshold": 50000000,
    "max_concurrency": 3,
    "multipart_chunksize": 8388608,
    "parallel_uploads": 2,
    "max_retries": 3,
    "exclude_patterns": ["*.tmp", "~$*", ".git/*"],
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "契約書",
      "source": "documents/contracts/",
      "bucket": "legal-documents",
      "s3_key_prefix": "contracts/",
      "recursive": true,
      "enabled": true
    },
    {
      "name": "提案書",
      "source": "documents/proposals/",
      "bucket": "legal-documents",
      "s3_key_prefix": "proposals/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

## 環境別設定

### 1. 開発環境

```json
{
  "logging": {
    "level": "DEBUG",
    "format": "%(asctime)s - %(name)s - %(levelname)s - %(funcName)s:%(lineno)d - %(message)s"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "dev"
  },
  "options": {
    "parallel_uploads": 1,
    "max_retries": 1,
    "dry_run": true,
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "開発用テストファイル",
      "source": "test-data/",
      "bucket": "dev-test-bucket",
      "s3_key_prefix": "test/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 2. ステージング環境

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(levelname)s - %(message)s",
    "file": "logs/staging.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "staging",
    "assume_role": {
      "role_arn": "arn:aws:iam::123456789012:role/StagingRole",
      "session_name": "staging-uploader"
    }
  },
  "options": {
    "parallel_uploads": 2,
    "max_retries": 3,
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "ステージングデータ",
      "source": "staging-data/",
      "bucket": "staging-bucket",
      "s3_key_prefix": "data/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 3. 本番環境

```json
{
  "logging": {
    "level": "WARNING",
    "format": "%(asctime)s - %(levelname)s - %(message)s",
    "file": "logs/production.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "production",
    "assume_role": {
      "role_arn": "arn:aws:iam::123456789012:role/ProductionRole",
      "session_name": "production-uploader",
      "duration_seconds": 3600
    }
  },
  "options": {
    "multipart_threshold": 100000000,
    "max_concurrency": 4,
    "multipart_chunksize": 10485760,
    "parallel_uploads": 3,
    "max_retries": 5,
    "timeout_seconds": 600,
    "enable_progress": false
  },
  "upload_tasks": [
    {
      "name": "本番データバックアップ",
      "source": "production-data/",
      "bucket": "production-backup",
      "s3_key_prefix": "backups/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

## 除外パターンの例

### 1. 一般的な除外パターン

```json
{
  "options": {
    "exclude_patterns": [
      "*.tmp",
      "*.log",
      "*.cache",
      ".DS_Store",
      "Thumbs.db",
      "*.swp",
      "*~"
    ]
  }
}
```

### 2. 開発環境用除外パターン

```json
{
  "options": {
    "exclude_patterns": [
      ".git/*",
      "node_modules/*",
      "*.pyc",
      "__pycache__/*",
      ".env",
      ".venv/*",
      "venv/*",
      "*.egg-info/*"
    ]
  }
}
```

### 3. システムファイル除外パターン

```json
{
  "options": {
    "exclude_patterns": [
      "System Volume Information/*",
      "$RECYCLE.BIN/*",
      ".Trashes/*",
      ".Spotlight-V100/*",
      ".fseventsd/*",
      ".DocumentRevisions-V100/*"
    ]
  }
}
```

## 設定の組み合わせ例

### 1. 完全な設定例

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    "file": "logs/s3_uploader.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "default",
    "assume_role": {
      "role_arn": "arn:aws:iam::123456789012:role/S3UploaderRole",
      "session_name": "s3-uploader-session",
      "duration_seconds": 3600
    }
  },
  "options": {
    "multipart_threshold": 100000000,
    "max_concurrency": 4,
    "multipart_chunksize": 10485760,
    "use_threads": true,
    "max_io_queue": 100,
    "io_chunksize": 262144,
    "exclude_patterns": [
      "*.tmp",
      "*.log",
      ".DS_Store",
      "Thumbs.db"
    ],
    "dry_run": false,
    "max_retries": 3,
    "timeout_seconds": 300,
    "parallel_uploads": 2,
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "重要な設定ファイル",
      "description": "システム設定のバックアップ",
      "source": "config/system.conf",
      "bucket": "config-backup",
      "s3_key": "backups/system.conf",
      "enabled": true
    },
    {
      "name": "ユーザーデータ",
      "description": "ユーザーアップロードファイル",
      "source": "uploads/users/",
      "bucket": "user-data",
      "s3_key_prefix": "user-uploads/",
      "recursive": true,
      "enabled": true
    },
    {
      "name": "アプリケーションログ",
      "description": "アプリケーションログのアーカイブ",
      "source": "logs/app/",
      "bucket": "log-archive",
      "s3_key_prefix": "archived-logs/",
      "recursive": true,
      "enabled": false
    }
  ]
}
```

## 設定のベストプラクティス

### 1. 環境変数の活用

設定ファイルにて環境変数を参照することで、環境別の設定を管理できます：

```json
{
  "aws": {
    "region": "${AWS_REGION}",
    "profile": "${AWS_PROFILE}"
  },
  "upload_tasks": [
    {
      "name": "環境別アップロード",
      "source": "${DATA_DIR}",
      "bucket": "${S3_BUCKET}",
      "s3_key_prefix": "${S3_PREFIX}",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 2. 設定の分離

大きな設定ファイルは用途別に分離することを推奨します：

- `config-dev.json`: 開発環境用
- `config-staging.json`: ステージング環境用
- `config-prod.json`: 本番環境用

### 3. セキュリティの考慮

- 認証情報は設定ファイルに含めない
- AWSプロファイルやAssumeRoleを活用
- 設定ファイルの適切な権限設定

これらの設定例を参考に、あなたの環境や要件に合わせた設定を作成してください。