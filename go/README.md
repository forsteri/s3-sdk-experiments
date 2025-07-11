# S3 Uploader - Go版

Python版のS3 UploaderをGoで再実装したプロジェクトです。

## プロジェクト構造

```
go/
├── go.mod              # Goモジュール定義
├── go.sum              # 依存関係のチェックサム（自動生成）
├── main.go             # エントリーポイント
├── cmd/                # アプリケーションのメインコマンド
│   ├── task-runner/    # タスクランナー（実装済み）
│   │   └── main.go
│   ├── scan-test/      # ファイルスキャンテスト
│   │   └── main.go
│   ├── client-test/    # S3クライアントテスト
│   │   └── main.go
│   └── upload-test/    # アップロードテスト
│       └── main.go
├── internal/           # 内部パッケージ（外部から使用不可）
│   ├── models/         # 設定管理（実装済み）
│   │   └── config.go
│   ├── logger/         # ログ管理（実装済み）
│   │   └── logger.go
│   ├── fileutils/      # ファイルスキャン（実装済み）
│   │   ├── scanner.go
│   │   ├── scanner_test.go
│   │   └── scanner_bench_test.go
│   ├── aws/            # AWS関連（実装済み）
│   │   ├── client.go   # S3クライアント管理
│   │   └── operations.go # S3操作ヘルパー
│   ├── uploader/       # アップロード処理（実装済み）
│   │   ├── uploader.go # 基本的なアップロード機能
│   │   ├── retry.go    # リトライ機能
│   │   └── task_runner.go # タスクランナー
│   └── progress/       # 進捗管理（未実装）
├── pkg/                # 外部パッケージ（ライブラリとして利用可能）
├── config.json         # 設定ファイル
├── logs/               # ログ出力ディレクトリ
└── test_fileutils.sh   # ファイルスキャンテスト実行スクリプト
```

## 開発の進め方

1. **設定管理**: JSONファイルの読み込みと構造体へのマッピング ✅
2. **AWS接続**: SDK v2を使ったS3クライアントの作成 ✅
3. **ファイル操作**: ファイルスキャンとアップロード準備 ✅
4. **並列処理**: goroutinesを使った並列アップロード 🚧
5. **進捗表示**: リアルタイム進捗管理 🚧
6. **エラーハンドリング**: リトライとログ出力 🚧

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

### タスクランナー機能 (internal/uploader/task_runner.go)
- config.jsonのupload_tasksを自動実行
- 個別タスクの実行もサポート
- 実行結果の詳細レポート
- ドライランモード対応
- 失敗時の適切な終了コード

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

## 次のステップ

基本的なアップローダー機能は完成しました！
残りは高機能化の実装：
- 並列アップロード機能 (goroutines + worker pool)
- 進捗表示機能 (internal/progress)
- マルチパートアップロード対応（大容量ファイル向け）
