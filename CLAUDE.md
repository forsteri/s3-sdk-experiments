# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

S3 SDK実験プロジェクト - 並列処理、リトライ機能、進捗追跡を持つ高度なS3アップローダーの実装

## 主要コマンド

### 開発環境セットアップ
```bash
uv sync                    # 依存関係のインストール
```

### アプリケーション実行
```bash
uv run python main.py      # メインアプリケーションの実行
```

### テスト実行
```bash
python test_*.py           # 各テストファイルの実行
```

### インフラ管理
```bash
terraform init             # Terraformの初期化
terraform apply            # AWSリソースの作成
terraform destroy          # AWSリソースの削除
```

## アーキテクチャ

### コア設計パターン
- **クラスベース設計**: 機能ごとに責任を分離（S3Uploader, TaskRunner, UploadExecutor）
- **並列処理**: ThreadPoolExecutorを使用した効率的なファイルアップロード
- **設定駆動**: JSONファイルによる柔軟な設定管理

### モジュール構造
```
src/
├── core/           # コアS3アップロード機能
│   ├── s3_uploader.py      # メインアップローダークラス
│   ├── task_runner.py      # タスク実行管理
│   └── upload_executor.py  # 並列アップロード実行
├── models/         # データモデル
│   └── upload_task.py      # アップロードタスクモデル
└── utils/          # ユーティリティ
    └── logger.py           # ログ設定
```

### 重要なクラス関係
- `S3Uploader`: メインの調整クラス、設定管理とタスク実行を統合
- `TaskRunner`: アップロードタスクの作成と管理
- `ParallelUploadExecutor`: 並列アップロード処理の実装
- `UploadTask`: アップロードタスクのデータモデル

## 設定管理

### 設定ファイル構造
```json
{
  "s3": {
    "bucket_name": "バケット名",
    "region": "リージョン",
    "profile_name": "AWSプロファイル名"
  },
  "upload": {
    "source_directory": "アップロード元ディレクトリ",
    "target_prefix": "S3のプレフィックス",
    "max_workers": 並列数,
    "retry_count": リトライ回数
  }
}
```

## AWS認証

### サポートされる認証方法
1. **AWSプロファイル**: `~/.aws/credentials`のプロファイル使用
2. **AssumeRole**: 別のロールを引き受けて実行
3. **デフォルト認証**: 環境変数やEC2ロールなど

### 設定例
```json
{
  "s3": {
    "profile_name": "my-profile",
    "assume_role_arn": "arn:aws:iam::123456789012:role/MyRole"
  }
}
```

## 重要な開発情報

### エラーハンドリング
- 詳細なエラー情報とスタックトレースの記録
- リトライ機能による一時的な障害の自動復旧
- 失敗したタスクの個別追跡

### 進捗追跡
- アップロード進捗のリアルタイム表示
- 成功/失敗の統計情報
- 詳細な実行ログ

### テストデータ
- `test_data/`ディレクトリ内のファイルを使用
- 各テストは独立したテストデータを持つ
- Terraformでテスト用バケットを作成可能