# S3 Uploader API リファレンス

## 概要

このドキュメントでは、S3 Uploaderの主要なクラスとメソッドについて詳しく説明します。

## メインクラス

### TaskRunner

アップロードタスクの実行を管理するメインクラスです。このクラスが全体の処理を統括します。

#### 初期化

```python
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.models.config import Config

config = Config.from_file("config.json")
runner = TaskRunner(config)
```

**パラメータ**:
- `config` (Config): 設定オブジェクト

#### メソッド

##### `run_all_tasks() -> Tuple[int, int]`

アップロードタスクを実行します。

**戻り値**:
- `Tuple[int, int]`: (成功したタスク数, 失敗したタスク数)

**例**:
```python
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.models.config import Config
from s3_uploader.utils.logger import LoggerManager

config = Config.from_file("config.json")
LoggerManager.setup(config.logging)
runner = TaskRunner(config)
successful, failed = runner.run_all_tasks()
print(f"成功: {successful}, 失敗: {failed}")
```

---

## 設定管理クラス

### Config

設定ファイルの読み込みと管理を行います。

#### クラスメソッド

##### `Config.from_file(config_path: str) -> Config`

設定ファイルからConfigオブジェクトを作成します。

**パラメータ**:
- `config_path` (str): 設定ファイルのパス

**戻り値**:
- `Config`: 設定オブジェクト

**例外**:
- `FileNotFoundError`: 設定ファイルが見つからない場合
- `ValueError`: JSON形式が不正な場合
- `RuntimeError`: その他の設定読み込みエラー

**例**:
```python
from s3_uploader.models.config import Config

config = Config.from_file("config.json")
```

#### 属性

- `logging` (LoggingConfig): ログ設定
- `aws` (AWSConfig): AWS設定
- `options` (UploadOptions): アップロードオプション
- `upload_tasks` (List[UploadTask]): アップロードタスクのリスト

---

## データクラス

### LoggingConfig

ログ設定を管理するデータクラスです。

```python
@dataclass
class LoggingConfig:
    level: str = "INFO"
    format: str = "%(asctime)s - %(levelname)s - %(message)s"
    file: Optional[str] = None
```

**属性**:
- `level` (str): ログレベル（DEBUG, INFO, WARNING, ERROR, CRITICAL）
- `format` (str): ログメッセージの形式
- `file` (Optional[str]): ログファイルの出力先

### AWSConfig

AWS関連の設定を管理するデータクラスです。

```python
@dataclass
class AWSConfig:
    region: str
    profile: Optional[str] = None
    assume_role: Optional[AssumeRoleConfig] = None
```

**属性**:
- `region` (str): AWSリージョン
- `profile` (Optional[str]): AWSプロファイル名
- `assume_role` (Optional[AssumeRoleConfig]): AssumeRole設定

### AssumeRoleConfig

AssumeRole設定を管理するデータクラスです。

```python
@dataclass
class AssumeRoleConfig:
    role_arn: str
    session_name: str
    external_id: Optional[str] = None
    duration_seconds: int = 3600
```

**属性**:
- `role_arn` (str): AssumeするロールのARN
- `session_name` (str): セッション名
- `external_id` (Optional[str]): 外部ID
- `duration_seconds` (int): セッションの有効期間（秒）

### UploadOptions

アップロードオプションを管理するデータクラスです。

```python
@dataclass
class UploadOptions:
    multipart_threshold: int = 100 * 1024 * 1024  # 100MB
    max_concurrency: int = 4
    multipart_chunksize: int = 10 * 1024 * 1024  # 10MB
    use_threads: bool = True
    max_io_queue: int = 100
    io_chunksize: int = 262144  # 256KB
    exclude_patterns: List[str] = field(default_factory=list)
    dry_run: bool = False
    max_retries: int = 3
    timeout_seconds: int = 300
    parallel_uploads: int = 2
    enable_progress: bool = True
```

**属性**:
- `multipart_threshold` (int): マルチパートアップロードの閾値
- `max_concurrency` (int): 同時並行数
- `multipart_chunksize` (int): マルチパートのチャンクサイズ
- `use_threads` (bool): スレッドプールの使用有無
- `max_io_queue` (int): I/Oキューの最大サイズ
- `io_chunksize` (int): I/Oチャンクサイズ
- `exclude_patterns` (List[str]): 除外するファイルパターン
- `dry_run` (bool): ドライランモード
- `max_retries` (int): リトライ回数
- `timeout_seconds` (int): タイムアウト時間
- `parallel_uploads` (int): 並列アップロード数
- `enable_progress` (bool): 進捗表示の有効/無効

### UploadTask

個別のアップロードタスクを管理するデータクラスです。

```python
@dataclass
class UploadTask:
    name: str
    source: str
    bucket: str
    description: Optional[str] = None
    enabled: bool = True
    s3_key: Optional[str] = None
    s3_key_prefix: Optional[str] = None
    recursive: bool = False
```

**属性**:
- `name` (str): タスク名
- `source` (str): アップロード元のパス
- `bucket` (str): アップロード先のS3バケット名
- `description` (Optional[str]): タスクの説明
- `enabled` (bool): タスクの有効/無効
- `s3_key` (Optional[str]): 単一ファイルの場合のS3キー
- `s3_key_prefix` (Optional[str]): ディレクトリの場合のS3キープレフィックス
- `recursive` (bool): ディレクトリを再帰的にアップロードするか

---

## コアクラス

### TaskRunner

アップロードタスクの実行を管理するクラスです。

#### 初期化

```python
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.models.config import Config

config = Config.from_file("config.json")
runner = TaskRunner(config)
```

**パラメータ**:
- `config` (Config): 設定オブジェクト

#### メソッド

##### `run_all_tasks() -> Tuple[int, int]`

すべてのアップロードタスクを実行します。

**戻り値**:
- `Tuple[int, int]`: (成功したタスク数, 失敗したタスク数)

### S3ClientManager

S3クライアントの作成と管理を行うクラスです。

#### 初期化

```python
from s3_uploader.core.s3_client import S3ClientManager
from s3_uploader.models.config import AWSConfig

aws_config = AWSConfig(region="ap-northeast-1")
manager = S3ClientManager(aws_config)
```

**パラメータ**:
- `aws_config` (AWSConfig): AWS設定オブジェクト

#### メソッド

##### `get_client() -> boto3.client`

S3クライアントを取得します（必要に応じて作成）。

**戻り値**:
- `boto3.client`: S3クライアント

**例外**:
- `NoCredentialsError`: AWS認証情報が見つからない場合
- `Exception`: その他のクライアント作成エラー

##### `_assume_role() -> Optional[Dict[str, str]]`

AssumeRoleを実行して一時的な認証情報を取得します。

**戻り値**:
- `Optional[Dict[str, str]]`: 一時的な認証情報（失敗時はNone）

### UploadExecutor

ファイルアップロードの実行を管理するクラスです。

#### 初期化

```python
from s3_uploader.core.uploader import UploadExecutor
from s3_uploader.models.config import UploadOptions

options = UploadOptions()
executor = UploadExecutor(s3_client, options)
```

**パラメータ**:
- `s3_client`: boto3のS3クライアント
- `options` (UploadOptions): アップロードオプション

#### メソッド

##### `upload_file(file_info: FileInfo, bucket: str, s3_key: str) -> UploadResult`

単一ファイルをアップロードします。

**パラメータ**:
- `file_info` (FileInfo): ファイル情報
- `bucket` (str): アップロード先のS3バケット名
- `s3_key` (str): S3キー

**戻り値**:
- `UploadResult`: アップロード結果

##### `_retry_upload(file_info: FileInfo, bucket: str, s3_key: str) -> UploadResult`

リトライ機能付きのアップロードを実行します。

**パラメータ**:
- `file_info` (FileInfo): ファイル情報
- `bucket` (str): アップロード先のS3バケット名
- `s3_key` (str): S3キー

**戻り値**:
- `UploadResult`: アップロード結果

### ParallelUploadExecutor

並列アップロードの実行を管理するクラスです。

#### 初期化

```python
from s3_uploader.core.uploader import ParallelUploadExecutor

parallel_executor = ParallelUploadExecutor(executor, max_workers=2)
```

**パラメータ**:
- `executor` (UploadExecutor): アップロード実行クラス
- `max_workers` (int): 最大ワーカー数

#### メソッド

##### `upload_files(upload_tasks: List[Tuple[FileInfo, str, str]]) -> Tuple[int, int]`

複数ファイルを並列でアップロードします。

**パラメータ**:
- `upload_tasks` (List[Tuple[FileInfo, str, str]]): (FileInfo, bucket, s3_key) のタプルのリスト

**戻り値**:
- `Tuple[int, int]`: (成功数, 失敗数)

---

## 結果クラス

### UploadResult

アップロード結果を表すデータクラスです。

```python
@dataclass
class UploadResult:
    file_path: str
    success: bool
    error: Optional[str] = None
```

**属性**:
- `file_path` (str): アップロードしたファイルのパス
- `success` (bool): アップロードが成功したか
- `error` (Optional[str]): エラーメッセージ（失敗時）

---

## ユーティリティクラス

### LoggerManager

ログ設定の管理を行うクラスです。

#### クラスメソッド

##### `LoggerManager.setup(config: LoggingConfig) -> logging.Logger`

ロガーをセットアップします。

**パラメータ**:
- `config` (LoggingConfig): ログ設定

**戻り値**:
- `logging.Logger`: 設定されたロガー

##### `LoggerManager.get_logger() -> logging.Logger`

設定済みのロガーを取得します。

**戻り値**:
- `logging.Logger`: 設定されたロガー

**例外**:
- `RuntimeError`: ロガーが初期化されていない場合

### FileScanner

ファイルスキャンを行うユーティリティクラスです。

#### 初期化

```python
from s3_uploader.utils.file_utils import FileScanner

scanner = FileScanner(exclude_patterns=["*.tmp", "*.log"])
```

**パラメータ**:
- `exclude_patterns` (List[str]): 除外するファイルパターン

#### メソッド

##### `scan_directory(directory: str, recursive: bool = False) -> List[FileInfo]`

ディレクトリをスキャンしてファイル情報を取得します。

**パラメータ**:
- `directory` (str): スキャン対象のディレクトリパス
- `recursive` (bool): 再帰的にスキャンするか

**戻り値**:
- `List[FileInfo]`: ファイル情報のリスト

##### `get_file_info(file_path: str) -> FileInfo`

単一ファイルの情報を取得します。

**パラメータ**:
- `file_path` (str): ファイルパス

**戻り値**:
- `FileInfo`: ファイル情報

---

## 使用例

### 基本的な使用例

```python
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.models.config import Config
from s3_uploader.utils.logger import LoggerManager

# 設定を読み込み
config = Config.from_file("config.json")

# ログを設定
LoggerManager.setup(config.logging)

# TaskRunnerを初期化
runner = TaskRunner(config)

# アップロードを実行
successful, failed = runner.run_all_tasks()

# 結果を表示
print(f"成功: {successful}, 失敗: {failed}")
```

### 設定のカスタマイズ

```python
from s3_uploader.models.config import Config, LoggingConfig, AWSConfig, UploadOptions, UploadTask

# 設定を手動で作成
config = Config(
    logging=LoggingConfig(level="DEBUG"),
    aws=AWSConfig(region="ap-northeast-1", profile="my-profile"),
    options=UploadOptions(parallel_uploads=4, dry_run=True),
    upload_tasks=[
        UploadTask(
            name="テストアップロード",
            source="test.txt",
            bucket="my-bucket",
            s3_key="test.txt"
        )
    ]
)

# TaskRunnerを直接使用
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.utils.logger import LoggerManager

LoggerManager.setup(config.logging)
runner = TaskRunner(config)
successful, failed = runner.run_all_tasks()
```

### エラーハンドリング

```python
from s3_uploader.core.task_runner import TaskRunner
from s3_uploader.models.config import Config
from s3_uploader.utils.logger import LoggerManager

try:
    # 設定ファイルを読み込み
    config = Config.from_file("config.json")
    
    # ログを設定
    LoggerManager.setup(config.logging)
    
    # アップロードを実行
    runner = TaskRunner(config)
    successful, failed = runner.run_all_tasks()
    
    if failed > 0:
        print(f"警告: {failed}件のアップロードが失敗しました")
    else:
        print("すべてのアップロードが成功しました")
        
except FileNotFoundError as e:
    print(f"設定ファイルが見つかりません: {e}")
except ValueError as e:
    print(f"設定ファイルの形式が正しくありません: {e}")
except Exception as e:
    print(f"予期しないエラーが発生しました: {e}")
```