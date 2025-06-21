# S3 SDK実験プロジェクト

AWSのS3サービスを使用したファイルアップロード機能の実装実験プロジェクトです。boto3 SDKを使用してローカルファイルをS3バケットにアップロードする基本的な操作を学習・検証します。

## プロジェクト構造

```
python/
├── README.md          # このファイル
├── main.py           # メインスクリプト
├── pyproject.toml    # プロジェクト設定とパッケージ依存関係
└── uv.lock          # 依存関係のロックファイル
```

## main.pyについて

このスクリプトは、boto3を利用してローカルファイルをS3バケットにアップロードするサンプルです。

### セットアップ

1. **uvのインストール**（まだインストールしていない場合）:
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   ```

2. **依存関係のインストール**:
   ```bash
   uv sync
   ```

### AWS認証情報の設定

以下のいずれかの方法でAWS認証情報を設定してください：

1. **AWS CLIを使用**:
   ```bash
   aws configure
   ```

2. **環境変数を設定**:
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   export AWS_DEFAULT_REGION=ap-northeast-1
   ```

3. **認証情報ファイルを直接編集** (`~/.aws/credentials`):
   ```ini
   [default]
   aws_access_key_id = your_access_key
   aws_secret_access_key = your_secret_key
   region = ap-northeast-1
   ```

### 実行方法

```bash
uv run python main.py
```

または仮想環境をアクティベートしてから実行:

```bash
source .venv/bin/activate  # Linux/macOS
python main.py
```

**動作内容**:
- `../test-data/sample_data.csv` を `s3-experiment-bucket-250615` バケットの `sample_data.csv` としてアップロード
- アップロードの成否がコンソールに表示

### 依存パッケージ

- **boto3**: AWS SDK for Python
- **botocore**: boto3の低レベルインターフェース

パッケージ管理にはuvを使用しています。`pyproject.toml`で依存関係を管理し、`uv.lock`でバージョンを固定しています。

## トラブルシューティング

### よくあるエラーと対処法

**1. `NoCredentialsError`**
```
botocore.exceptions.NoCredentialsError: Unable to locate credentials
```
**対処法**: AWS認証情報が正しく設定されていません。上記の「AWS認証情報の設定」セクションを参照して設定してください。

**2. `ClientError: NoSuchBucket`**
```
botocore.exceptions.ClientError: An error occurred (NoSuchBucket) when calling the PutObject operation
```
**対処法**: 
- S3バケットが存在しない、または名前が間違っています
- `main.py`内のバケット名を確認してください
- 必要に応じてTerraformでバケットを作成してください

**3. `AccessDenied`**
```
botocore.exceptions.ClientError: An error occurred (AccessDenied) when calling the PutObject operation
```
**対処法**: 
- AWS認証情報に適切な権限がありません
- S3への書き込み権限（`s3:PutObject`）が必要です
- IAMポリシーを確認してください

**4. `FileNotFoundError`**
```
FileNotFoundError: [Errno 2] No such file or directory: '../test-data/sample_data.csv'
```
**対処法**: 
- アップロード対象のファイルが存在しません
- ファイルパスが正しいか確認してください
- プロジェクトルートから実行されているか確認してください

### デバッグのヒント

- **ログの有効化**: boto3のログを有効にすると詳細なエラー情報が表示されます
- **ファイルパスの確認**: 相対パスではなく絶対パスを使用することを検討してください
- **権限の確認**: `aws s3 ls`コマンドでS3への接続と権限を事前に確認してください