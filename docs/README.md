# S3 Uploader - 初級者向けガイド

## 概要

S3 Uploaderは、Amazon S3（Simple Storage Service）にファイルを効率的にアップロードするためのPythonツールです。このツールは特に大量のファイルを扱う際に有用で、並列処理、リトライ機能、進捗追跡などの高度な機能を提供します。

## 主な特徴

- **並列アップロード**: 複数のファイルを同時にアップロードして処理時間を短縮
- **リトライ機能**: ネットワーク障害などの一時的な問題に対して自動的に再試行
- **進捗追跡**: アップロードの進捗をリアルタイムで表示
- **柔軟な設定**: JSON設定ファイルで様々なオプションを制御
- **AWS認証**: プロファイル認証やAssumeRoleなど複数の認証方式に対応

## インストールと環境セットアップ

### 必要な要件

- Python 3.8以上
- AWS CLI（設定済み）
- boto3ライブラリ

### セットアップ手順

1. **依存関係のインストール**
   ```bash
   uv sync
   ```

2. **AWS認証情報の設定**
   ```bash
   aws configure
   ```
   または、`~/.aws/credentials`ファイルに認証情報を設定

3. **設定ファイルの作成**
   `config.json`ファイルを作成し、アップロード設定を定義

## 基本的な使い方

### 1. 設定ファイルの準備

`config.json`ファイルの例：

```json
{
  "logging": {
    "level": "INFO",
    "format": "%(asctime)s - %(levelname)s - %(message)s",
    "file": "logs/s3_uploader.log"
  },
  "aws": {
    "region": "ap-northeast-1",
    "profile": "default"
  },
  "options": {
    "multipart_threshold": 104857600,
    "max_concurrency": 4,
    "multipart_chunksize": 10485760,
    "parallel_uploads": 2,
    "max_retries": 3,
    "dry_run": false,
    "enable_progress": true
  },
  "upload_tasks": [
    {
      "name": "テストファイルのアップロード",
      "source": "test-data/sample.txt",
      "bucket": "my-test-bucket",
      "s3_key": "uploads/sample.txt",
      "enabled": true
    },
    {
      "name": "ディレクトリのアップロード",
      "source": "test-data/exports",
      "bucket": "my-test-bucket",
      "s3_key_prefix": "exports/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 2. アプリケーションの実行

```bash
uv run python main.py
```

## 設定項目の詳細説明

### ロギング設定（logging）

- **level**: ログレベル（DEBUG, INFO, WARNING, ERROR, CRITICAL）
- **format**: ログメッセージの形式
- **file**: ログファイルの出力先（省略可能）

### AWS設定（aws）

- **region**: AWS リージョン（例：ap-northeast-1）
- **profile**: AWS プロファイル名（省略可能）
- **assume_role**: AssumeRole設定（省略可能）

### アップロードオプション（options）

- **multipart_threshold**: マルチパートアップロードの閾値（バイト）
- **max_concurrency**: 同時並行数
- **multipart_chunksize**: マルチパートのチャンクサイズ（バイト）
- **parallel_uploads**: 並列アップロード数
- **max_retries**: リトライ回数
- **dry_run**: 実際のアップロードを行わないテストモード
- **enable_progress**: 進捗表示の有効/無効

### アップロードタスク（upload_tasks）

各タスクには以下の項目を設定できます：

- **name**: タスク名（識別用）
- **source**: アップロード元のファイルまたはディレクトリパス
- **bucket**: アップロード先のS3バケット名
- **s3_key**: 単一ファイルの場合のS3キー
- **s3_key_prefix**: ディレクトリの場合のS3キープレフィックス
- **recursive**: ディレクトリを再帰的にアップロードするか
- **enabled**: タスクの有効/無効

## よくある使用例

### 1. 単一ファイルのアップロード

```json
{
  "upload_tasks": [
    {
      "name": "重要なファイルのアップロード",
      "source": "/path/to/important-file.pdf",
      "bucket": "my-documents",
      "s3_key": "documents/important-file.pdf",
      "enabled": true
    }
  ]
}
```

### 2. ディレクトリ全体のアップロード

```json
{
  "upload_tasks": [
    {
      "name": "写真フォルダのバックアップ",
      "source": "/home/user/Pictures",
      "bucket": "my-backup-bucket",
      "s3_key_prefix": "backups/pictures/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

### 3. 複数のタスクを順番に実行

```json
{
  "upload_tasks": [
    {
      "name": "設定ファイルのアップロード",
      "source": "config/app.conf",
      "bucket": "my-config-bucket",
      "s3_key": "configs/app.conf",
      "enabled": true
    },
    {
      "name": "ログファイルのアップロード",
      "source": "logs/",
      "bucket": "my-log-bucket",
      "s3_key_prefix": "logs/",
      "recursive": true,
      "enabled": true
    }
  ]
}
```

## トラブルシューティング

### 認証エラー

**エラー**: `NoCredentialsError: Unable to locate credentials`

**解決方法**:
1. AWS CLIが正しく設定されているか確認
2. `aws configure`コマンドを実行して認証情報を設定
3. 環境変数`AWS_ACCESS_KEY_ID`と`AWS_SECRET_ACCESS_KEY`が設定されているか確認

### ファイルが見つからないエラー

**エラー**: `FileNotFoundError: Source file not found`

**解決方法**:
1. 設定ファイルの`source`パスが正しいか確認
2. ファイルやディレクトリが存在するか確認
3. パス区切り文字が正しいか確認（Windows: `\`, Unix: `/`）

### アップロード失敗

**エラー**: `ClientError: Access Denied`

**解決方法**:
1. S3バケットへの書き込み権限があるか確認
2. バケット名が正しいか確認
3. リージョンが正しいか確認

## パフォーマンスの最適化

### 並列処理の調整

- **parallel_uploads**: 同時にアップロードするファイル数
- **max_concurrency**: 単一ファイルの並列度
- 値を大きくすると処理速度が向上しますが、メモリ使用量も増加します

### リトライ設定の調整

- **max_retries**: ネットワーク障害時の再試行回数
- 不安定な接続環境では値を大きくすることを推奨

### ログレベルの調整

- デバッグ時：`DEBUG`
- 通常運用時：`INFO`
- 最小限のログ：`WARNING`

## より詳細な情報

- [API リファレンス](api-reference.md)
- [設定例集](configuration-examples.md)
- [アーキテクチャ概要](architecture.md)