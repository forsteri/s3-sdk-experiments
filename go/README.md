# S3 Uploader - Go版

Python版のS3 UploaderをGoで再実装したプロジェクトです。

## プロジェクト構造

```
go/
├── go.mod              # Goモジュール定義
├── go.sum              # 依存関係のチェックサム（自動生成）
├── main.go             # エントリーポイント
├── cmd/                # アプリケーションのメインコマンド
│   ├── uploader/
│   │   └── main.go
│   └── scan-test/      # ファイルスキャンテスト
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
│   ├── aws/            # AWS関連（未実装）
│   ├── uploader/       # アップロード処理（未実装）
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

## 次のステップ

アップロード処理の実装に進みます。以下の機能を追加予定：
- S3クライアントマネージャー (internal/aws)
- アップロード実行機能 (internal/uploader)
- 進捗表示機能 (internal/progress)
- タスクランナー (internal/uploader)
