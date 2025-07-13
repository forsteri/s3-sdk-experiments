# S3 Uploader - Go版

Python版のS3 UploaderをGoで再実装したプロジェクトです。**シングルバイナリで動作**し、設定ファイルベースで複数のアップロードタスクを自動実行できます。

## クイックスタート

```bash
# ビルド
make build

# 実行（config.jsonの全タスクを実行）
./bin/s3-uploader

# バージョン確認
./bin/s3-uploader --version

# ドライランモード
./bin/s3-uploader --dry-run

# テストモード（単一ファイルアップロード）
./bin/s3-uploader --test
```

## 主な機能

- 📁 単一ファイル/ディレクトリの S3 アップロード
- 🚀 並列アップロード機能（ワーカープール方式）
- 📦 マルチパートアップロード（大容量ファイル対応）
- ⚡ 並列マルチパートアップロード（高速転送）
- 🔄 自動リトライ機能（指数バックオフ）
- 📊 進捗表示機能（ログ出力対応）
- 🏃 ドライランモード
- 🎯 除外パターンのサポート
- 📋 タスクランナー（設定ベースの自動実行）
- 🔐 AssumeRole対応

## プロジェクト構造

```
go/
├── Makefile            # ビルド・開発タスク管理
├── go.mod              # Goモジュール定義
├── go.sum              # 依存関係のチェックサム
├── main.go             # メインエントリーポイント（s3-uploader）
├── config.json         # 設定ファイル
├── cmd/                # 個別ツール（開発・デバッグ用）
│   ├── task-runner/    # タスクランナー単体（main.goと同等機能）
│   ├── scan-test/      # ファイルスキャンテスト
│   ├── client-test/    # S3クライアントテスト
│   ├── upload-test/    # アップロードテスト
│   ├── parallel-test/  # 並列アップロードテスト
│   └── multipart-test/ # マルチパートアップロードテスト
├── internal/           # 内部パッケージ
│   ├── models/         # 設定管理
│   ├── logger/         # ログ管理
│   ├── version/        # バージョン情報
│   ├── fileutils/      # ファイルスキャン
│   ├── aws/            # AWS/S3関連
│   ├── uploader/       # アップロード処理
│   └── progress/       # 進捗管理
├── scripts/            # テスト・ユーティリティスクリプト
└── bin/                # ビルド成果物（.gitignore）
    └── s3-uploader     # メインバイナリ（シングルバイナリ）
```

**注**: `s3-uploader`バイナリ1つで全機能が利用可能です。`cmd/`配下のツールは開発・デバッグ用の個別機能です。

## 開発の進め方

1. **設定管理**: JSONファイルの読み込みと構造体へのマッピング ✅
2. **AWS接続**: SDK v2を使ったS3クライアントの作成 ✅
3. **ファイル操作**: ファイルスキャンとアップロード準備 ✅
4. **並列処理**: goroutinesを使った並列アップロード ✅
5. **進捗表示**: リアルタイム進捗管理 ✅
6. **エラーハンドリング**: リトライとログ出力 ✅

## 実装済み機能

### ファイルスキャン機能 (internal/fileutils)
- Python版と同等の機能を実装
- 除外パターンのサポート（*.tmp, .DS_Store など）
- 再帰的/非再帰的ディレクトリスキャン
- 単一ファイル情報の取得
- 包括的なユニットテストとベンチマークテスト

### AWS S3クライアント管理 (internal/aws)
- 設定ベースのS3クライアント作成
- プロファイル認証とAssumeRole対応
- リトライ設定のカスタマイズ
- S3操作ヘルパー関数（アップロード、一覧取得、存在確認）
- メタデータ付きアップロード
- Content-Typeの自動推測

### ファイルアップロード機能 (internal/uploader)
- 単一ファイルアップロード
- ディレクトリアップロード（再帰的/非再帰的）
- ドライランモード対応
- リトライ機能（指数バックオフ）
- 除外パターンの適用
- 詳細なアップロード結果レポート

### 並列アップロード機能 (internal/uploader/parallel.go)
- ワーカープール方式による効率的な並列処理
- 設定可能なワーカー数（config.jsonのparallel_uploads）
- ファイル数に応じた自動的な並列/順次処理の切り替え
- 各ワーカーが独立してリトライ処理を実行
- リアルタイムの統計情報追跡（アップロード数、失敗数、総バイト数）
- コンテキストによる適切なキャンセル処理
- 進捗追跡機能との統合

### 進捗表示機能 (internal/progress)
- サーバー環境向けのログベース進捗表示（デフォルト）
- 定期的な進捗レポート（30秒間隔、設定変更可能）
- ターミナルモード（プログレスバー表示）もサポート
- アクティブワーカーの状態追跡
- 転送速度とETA（推定残り時間）の計算
- atomic操作による安全な並列更新

### タスクランナー機能 (internal/uploader/task_runner.go)
- config.jsonのupload_tasksを自動実行
- 個別タスクの実行もサポート
- 実行結果の詳細レポート
- ドライランモード対応
- 失敗時の適切な終了コード
- 並列アップロードに対応

### マルチパートアップロード機能 (internal/aws/multipart.go, multipart_parallel.go)
- 大容量ファイルの効率的なアップロード（デフォルト: 100MB以上）
- ファイルを複数のパートに分割して並列転送
- 設定可能なチャンクサイズ（config.jsonのmultipart_chunksize）
- 順次・並列マルチパートアップロードの両方をサポート
- エラー時の自動クリーンアップ（AbortMultipartUpload）
- アップロード中の進捗追跡
- ワーカープール方式による効率的な並列処理
- ReadAt を使用したスレッドセーフなファイル読み込み

### 使用方法

1. **テストの実行**:
   ```bash
   chmod +x test_fileutils.sh
   ./test_fileutils.sh
   ```

2. **ファイルスキャンのテスト実行**:
   ```bash
   go run cmd/scan-test/main.go
   ```

3. **S3クライアントのテスト実行**:
   ```bash
   go run cmd/client-test/main.go
   # または特定のバケットを指定
   go run cmd/client-test/main.go -bucket your-bucket-name
   ```

4. **メインプログラムの実行**:
   ```bash
   # 通常モード（すべてのタスクを実行）
   go run main.go
   
   # テストモード（単一ファイルアップロードのテスト）
   go run main.go -test
   
   # ドライランモード
   go run main.go -dry-run
   ```

5. **タスクランナーの実行**:
   ```bash
   # すべてのタスクを実行
   go run cmd/task-runner/main.go
   
   # 特定のタスクのみを実行
   go run cmd/task-runner/main.go -task sample_data
   
   # ドライランモード
   go run cmd/task-runner/main.go -dry-run
   ```

6. **アップロードテストの実行**:
   ```bash
   # 単一ファイルのアップロード
   go run cmd/upload-test/main.go -source ../test-data/sample_data.csv -key test/sample.csv
   
   # ディレクトリのアップロード
   go run cmd/upload-test/main.go -source ../test-data/exports -key exports/ -recursive
   
   # ドライランモード
   go run cmd/upload-test/main.go -source ../test-data -dry-run -recursive
   ```

7. **並列アップロードのテスト実行**:
   ```bash
   # ディレクトリを並列アップロード（デフォルト: 3ワーカー）
   go run cmd/parallel-test/main.go -source ../test-data -recursive
   
   # ワーカー数を指定
   go run cmd/parallel-test/main.go -source ../test-data -recursive -workers 8
   
   # 順次処理でアップロード（比較用）
   go run cmd/parallel-test/main.go -source ../test-data -recursive -parallel=false
   
   # ベンチマークモード（並列 vs 順次の性能比較）
   go run cmd/parallel-test/main.go -benchmark ../test-data
   ```

8. **進捗表示機能のテスト**:
   ```bash
   # テストスクリプトの実行
   chmod +x test_progress.sh
   ./test_progress.sh
   ```

9. **マルチパートアップロードのテスト実行**:
   ```bash
   # 大容量ファイルをマルチパートでアップロード
   go run cmd/multipart-test/main.go -source bigfile.zip -bucket your-bucket -key test/bigfile.zip
   
   # 強制的にマルチパートアップロードを使用（小さいファイルでも）
   go run cmd/multipart-test/main.go -source file.txt -bucket your-bucket -key test/file.txt -multipart
   
   # 並列マルチパートアップロード（ワーカー数指定）
   go run cmd/multipart-test/main.go -source bigfile.zip -bucket your-bucket -key test/bigfile.zip -workers 8
   
   # チャンクサイズを指定（MB単位）
   go run cmd/multipart-test/main.go -source bigfile.zip -bucket your-bucket -key test/bigfile.zip -chunk-size 10
   
   # ベンチマークモード（通常 vs 順次マルチパート vs 並列マルチパートの比較）
   go run cmd/multipart-test/main.go -source bigfile.zip -bucket your-bucket -key test/bigfile -benchmark
   
   # ドライランモード
   go run cmd/multipart-test/main.go -source bigfile.zip -bucket your-bucket -key test/bigfile.zip -dry-run
   ```

## 並列アップロードの特徴

- **自動最適化**: ファイル数が少ない場合は順次処理、多い場合は並列処理を自動選択
- **効率的なリソース管理**: ワーカープール方式でgoroutineを効率的に管理
- **リトライ対応**: 各ワーカーが独立してリトライ処理を実行
- **進捗追跡**: atomic操作でリアルタイムに進捗を追跡
- **ベンチマーク機能**: 異なるワーカー数での性能比較が可能

## 設定ファイル (config.json)

### 進捗表示の設定
```json
{
  "options": {
    "enable_progress": true,      // 進捗表示の有効/無効
    "parallel_uploads": 3,        // 並列ワーカー数（1で順次処理）
    // その他の設定...
  }
}
```

### 進捗ログの出力例
``` text
2024-03-15 10:30:00 - INFO - Upload progress - completed=150 total=500 processed=140 failed=5 skipped=5 bytes_processed=1073741824 speed_mbps=10.24 elapsed=1m40s eta=4m10s percentage=30.0%
```

## サーバー環境での利用

このツールはサーバー環境での定期実行を想定して設計されています：

- **Cron設定例**:
  ```bash
  # 毎日午前2時に実行
  0 2 * * * /path/to/go run /path/to/main.go >> /var/log/s3-uploader.log 2>&1
  ```

- **systemdサービス例**:
  ```ini
  [Unit]
  Description=S3 Uploader Service
  After=network.target

  [Service]
  Type=oneshot
  ExecStart=/usr/local/bin/go run /path/to/main.go
  StandardOutput=journal
  StandardError=journal

  [Timer]
  OnCalendar=daily
  Persistent=true

  [Install]
  WantedBy=timers.target
  ```

## 次のステップ

すべての主要機能が実装されました！✨
今後の拡張案：
- ✅ マルチパートアップロード対応（大容量ファイル向け）→ 実装済み！
- メトリクス出力（Prometheus形式）
- Slack/Email通知機能
- 差分アップロード機能
- ストリーミングアップロード（ファイルをメモリに読み込まずに転送）
