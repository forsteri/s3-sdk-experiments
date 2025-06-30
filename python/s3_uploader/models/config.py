"""設定管理用のデータクラス"""
from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any
import json
import os


@dataclass
class LoggingConfig:
    """ロギング設定"""
    level: str = "INFO"
    format: str = "%(asctime)s - %(levelname)s - %(message)s"
    file: Optional[str] = None


@dataclass
class AssumeRoleConfig:
    """AssumeRole設定"""
    role_arn: str
    session_name: str
    external_id: Optional[str] = None
    duration_seconds: int = 3600


@dataclass
class AWSConfig:
    """AWS関連の設定"""
    region: str
    profile: Optional[str] = None
    assume_role: Optional[AssumeRoleConfig] = None

    def __post_init__(self):
        if self.assume_role and isinstance(self.assume_role, dict):
            self.assume_role = AssumeRoleConfig(**self.assume_role)


@dataclass
class UploadOptions:
    """アップロードオプション"""
    multipart_threshold: int = 100 * 1024 * 1024  # 100MB
    max_concurrency: int = 4
    multipart_chunksize: int = 10 * 1024 * 1024  # 10MB
    use_threads: bool = True
    max_io_queue: int = 100
    io_chunksize: int = 262144  # 256KB
    exclude_patterns: List[str] = field(default_factory=list)
    dry_run: bool = False
    max_retries: int = 3
    timeout_seconds: int = 300  # 追加
    parallel_uploads: int = 2
    enable_progress: bool = True


@dataclass
class UploadTask:
    """個別のアップロードタスク"""
    # 必須フィールド（デフォルト値なし）を先に
    name: str
    source: str
    bucket: str
    
    # オプションフィールド（デフォルト値あり）を後に
    description: Optional[str] = None  # 追加
    enabled: bool = True
    s3_key: Optional[str] = None  # ファイルの場合
    s3_key_prefix: Optional[str] = None  # ディレクトリの場合
    recursive: bool = False


@dataclass
class Config:
    """メイン設定クラス"""
    logging: LoggingConfig
    aws: AWSConfig
    options: UploadOptions
    upload_tasks: List[UploadTask]

    @classmethod
    def from_file(cls, config_path: str) -> 'Config':
        """設定ファイルから読み込み"""
        if not os.path.exists(config_path):
            raise FileNotFoundError(f"Configuration file {config_path} not found.")
        
        try:
            with open(config_path, "r", encoding="utf-8") as file:
                data = json.load(file)
            
            # 各セクションをパース
            logging_config = LoggingConfig(**data.get("logging", {}))
            aws_config = AWSConfig(**data.get("aws", {}))
            options = UploadOptions(**data.get("options", {}))
            
            # アップロードタスクをパース
            upload_tasks = [
                UploadTask(**task) for task in data.get("upload_tasks", [])
            ]
            
            return cls(
                logging=logging_config,
                aws=aws_config,
                options=options,
                upload_tasks=upload_tasks
            )
            
        except json.JSONDecodeError as e:
            raise ValueError(f"Error decoding JSON from {config_path}: {e}")
        except Exception as e:
            raise RuntimeError(f"Error loading configuration: {e}")