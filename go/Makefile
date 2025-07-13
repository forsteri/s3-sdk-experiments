.PHONY: help build clean test test-all test-fileutils test-benchmark test-integration test-multipart test-progress run run-dry task-runner

# デフォルトゴール
.DEFAULT_GOAL := help

# ヘルプ
help: ## ヘルプを表示
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# バージョン情報
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ビルドフラグ
LDFLAGS := -ldflags "-X 's3-uploader/internal/version.Version=$(VERSION)' -X 's3-uploader/internal/version.Commit=$(COMMIT)' -X 's3-uploader/internal/version.Date=$(DATE)'"

# ビルド
build: ## アプリケーションをビルド
	go build $(LDFLAGS) -o bin/s3-uploader main.go
	go build $(LDFLAGS) -o bin/task-runner cmd/task-runner/main.go
	@echo "✅ Build complete: bin/s3-uploader ($(VERSION))"

build-all: ## 全プラットフォーム向けにビルド
	@echo "Building for multiple platforms..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/s3-uploader-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/s3-uploader-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/s3-uploader-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/s3-uploader-windows-amd64.exe main.go
	@echo "✅ Multi-platform build complete"

install: ## バイナリをインストール（/usr/local/bin）
	go install $(LDFLAGS)
	@echo "✅ Installed to $(GOPATH)/bin/"

# クリーン
clean: ## ビルド成果物とテスト結果を削除
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f log.txt
	rm -rf logs/

# テスト関連
test: ## 全てのユニットテストを実行
	go test ./... -v

test-all: test test-fileutils test-benchmark test-integration test-multipart test-progress ## 全てのテスト（ユニット + 統合）を実行

test-fileutils: ## ファイルユーティリティのテストを実行
	@echo "=== Running File Utils Tests ==="
	@if [ -f scripts/test_fileutils.sh ]; then \
		chmod +x scripts/test_fileutils.sh && ./scripts/test_fileutils.sh; \
	else \
		go test ./internal/fileutils -v; \
	fi

test-benchmark: ## ベンチマークテストを実行
	@echo "=== Running Benchmark Tests ==="
	@if [ -f scripts/test_benchmark.sh ]; then \
		chmod +x scripts/test_benchmark.sh && ./scripts/test_benchmark.sh; \
	else \
		go test -bench=. ./...; \
	fi

test-integration: ## 統合テストを実行
	@echo "=== Running Integration Tests ==="
	@if [ -f scripts/test_integration.sh ]; then \
		chmod +x scripts/test_integration.sh && ./scripts/test_integration.sh; \
	else \
		echo "Integration test script not found"; \
	fi

test-multipart: ## マルチパートアップロードのテストを実行
	@echo "=== Running Multipart Upload Tests ==="
	@if [ -f scripts/test_multipart.sh ]; then \
		chmod +x scripts/test_multipart.sh && ./scripts/test_multipart.sh; \
	elif [ -f scripts/test_basic_multipart.sh ]; then \
		chmod +x scripts/test_basic_multipart.sh && ./scripts/test_basic_multipart.sh; \
	else \
		go run cmd/multipart-test/main.go -dry-run; \
	fi

test-progress: ## 進捗表示のテストを実行
	@echo "=== Running Progress Tests ==="
	@if [ -f scripts/test_progress.sh ]; then \
		chmod +x scripts/test_progress.sh && ./scripts/test_progress.sh; \
	else \
		echo "Progress test script not found"; \
	fi

test-coverage: ## カバレッジ付きでテストを実行
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 実行関連
run: ## メインプログラムを実行
	go run main.go

run-dry: ## ドライランモードで実行
	go run main.go -dry-run

run-test: ## テストモードで実行
	go run main.go -test

task-runner: ## タスクランナーを実行
	go run cmd/task-runner/main.go

task-runner-dry: ## タスクランナーをドライランモードで実行
	go run cmd/task-runner/main.go -dry-run

# 個別テストプログラムの実行
run-scan-test: ## ファイルスキャンテストを実行
	go run cmd/scan-test/main.go

run-client-test: ## S3クライアントテストを実行
	go run cmd/client-test/main.go

run-upload-test: ## アップロードテストを実行（要パラメータ）
	@echo "Usage: make run-upload-test ARGS='-source file.txt -key test/file.txt'"
	@echo "Running with: go run cmd/upload-test/main.go $(ARGS)"
	@go run cmd/upload-test/main.go $(ARGS)

run-parallel-test: ## 並列アップロードテストを実行（要パラメータ）
	@echo "Usage: make run-parallel-test ARGS='-source ../test-data -recursive'"
	@echo "Running with: go run cmd/parallel-test/main.go $(ARGS)"
	@go run cmd/parallel-test/main.go $(ARGS)

run-multipart-test: ## マルチパートアップロードテストを実行（要パラメータ）
	@echo "Usage: make run-multipart-test ARGS='-source bigfile.zip -bucket bucket -key test/bigfile.zip'"
	@echo "Running with: go run cmd/multipart-test/main.go $(ARGS)"
	@go run cmd/multipart-test/main.go $(ARGS)

# 開発用コマンド
fmt: ## コードをフォーマット
	go fmt ./...

lint: ## Lintを実行（要golangci-lint）
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Install it with:"; \
		echo "  brew install golangci-lint"; \
	fi

mod-tidy: ## go.modを整理
	go mod tidy

mod-update: ## 依存関係を更新
	go get -u ./...
	go mod tidy

# デバッグ
debug-multipart: ## マルチパートアップロードのデバッグ
	@if [ -f scripts/debug-multipart.sh ]; then \
		chmod +x scripts/debug-multipart.sh && ./scripts/debug-multipart.sh; \
	else \
		echo "Debug script not found"; \
	fi

check-bucket-region: ## バケットのリージョンをチェック
	@if [ -f scripts/check-bucket-region.sh ]; then \
		chmod +x scripts/check-bucket-region.sh && ./scripts/check-bucket-region.sh; \
	else \
		echo "Check bucket region script not found"; \
	fi
