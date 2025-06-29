import boto3
import json
import os
import logging
import fnmatch
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from botocore.exceptions import NoCredentialsError, ClientError
from boto3.s3.transfer import TransferConfig
from typing import Tuple, List

class ProgressTracker:
    """アップロード進捗を追跡するクラス"""
    
    def __init__(self, total_size: int, filename: str):
        self.total_size = total_size
        self.filename = filename
        self.uploaded_size = 0
        self.lock = threading.Lock()
        self.start_time = time.time()
        
    def __call__(self, bytes_transferred):
        """boto3のコールバック関数として使用"""
        with self.lock:
            self.uploaded_size += bytes_transferred
            progress = (self.uploaded_size / self.total_size) * 100
            elapsed_time = time.time() - self.start_time
            
            if elapsed_time > 0:
                speed = self.uploaded_size / elapsed_time / 1024 / 1024  # MB/s
                eta = (self.total_size - self.uploaded_size) / (self.uploaded_size / elapsed_time) if self.uploaded_size > 0 else 0
                
                print(f"\r{self.filename}: {progress:.1f}% ({self.uploaded_size}/{self.total_size}) - {speed:.2f} MB/s - ETA: {eta:.0f}s", end="", flush=True)
            
    def complete(self):
        """アップロード完了"""
        elapsed_time = time.time() - self.start_time
        speed = self.total_size / elapsed_time / 1024 / 1024 if elapsed_time > 0 else 0
        print(f"\r{self.filename}: Complete! - {speed:.2f} MB/s - {elapsed_time:.1f}s")


def setup_logging(config):
    """ロギングの設定を初期化する"""
    logging_config = config.get("logging", {})

    # デフォルト値
    level = logging_config.get("level", "INFO")
    format_str = logging_config.get(
        "format", "%(asctime)s - %(levelname)s - %(message)s"
    )
    log_file = logging_config.get("file", None)

    # ログレベルの設定
    log_level = getattr(logging, level.upper(), logging.INFO)

    # ロガーの設定
    handlers = []

    # コンソール出力
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(
        logging.Formatter(format_str, datefmt="%Y-%m-%d %H:%M:%S")
    )
    handlers.append(console_handler)

    if log_file:
        log_dir = os.path.dirname(log_file)
        if log_dir and not os.path.exists(log_dir):
            os.makedirs(log_dir)

        file_handler = logging.FileHandler(log_file, encoding="utf-8")
        file_handler.setFormatter(
            logging.Formatter(format_str, datefmt="%Y-%m-%d %H:%M:%S")
        )
        handlers.append(file_handler)

    # ロガーの設定
    logging.basicConfig(level=log_level, handlers=handlers)

    return logging.getLogger(__name__)


def load_config(config_path="config.json"):
    """設定ファイルを読み込む"""
    try:
        with open(config_path, "r", encoding="utf-8") as file:
            config = json.load(file)
        return config
    except FileNotFoundError:
        print(f"Configuration file {config_path} not found.")
        return None
    except json.JSONDecodeError as e:
        print(f"Error decoding JSON from the configuration file {config_path}: {e}")
        return None
    except Exception as e:
        print(f"An error occurred while loading the configuration: {e}")
        return None


def assume_role(aws_config, logger):
    """Assume Roleを実行して一時的な認証情報を取得する"""
    assume_role_config = aws_config.get("assume_role")
    if not assume_role_config:
        logger.error("Assume role configuration not found.")
        return None

    try:
        region = aws_config.get("region")
        endpoint_url = f"https://sts.{region}.amazonaws.com"
        profile = aws_config.get("profile")

        if profile:
            session = boto3.Session(profile_name=profile)
            sts_client = session.client(
                "sts", region_name=region, endpoint_url=endpoint_url
            )
        else:
            sts_client = boto3.client(
                "sts", region_name=region, endpoint_url=endpoint_url
            )

        # Assume Roleを実行
        role_arn = assume_role_config["role_arn"]
        session_name = assume_role_config["session_name"]
        external_id = assume_role_config.get("external_id")
        duration = assume_role_config.get("duration_seconds", 3600)

        assume_role_params = {
            "RoleArn": role_arn,
            "RoleSessionName": session_name,
            "DurationSeconds": duration,
        }

        if external_id:
            assume_role_params["ExternalId"] = external_id

        response = sts_client.assume_role(**assume_role_params)
        credentials = response["Credentials"]
        logger.info(f"Assumed role successfully: {role_arn}")
        return {
            "access_key_id": credentials["AccessKeyId"],
            "secret_access_key": credentials["SecretAccessKey"],
            "session_token": credentials["SessionToken"],
        }

    except ClientError as e:
        logger.error(f"An error occurred while assuming the role: {e}")
        return None
    except Exception as e:
        logger.error(f"An unexpected error occurred: {e}")
        return None


def create_s3_client(aws_config, logger):
    """S3クライアントを作成する"""
    try:
        region = aws_config.get("region")
        profile = aws_config.get("profile")

        temp_credentials = assume_role(aws_config, logger)
        if temp_credentials:
            s3_client = boto3.client(
                "s3",
                region_name=region,
                aws_access_key_id=temp_credentials["access_key_id"],
                aws_secret_access_key=temp_credentials["secret_access_key"],
                aws_session_token=temp_credentials["session_token"],
            )
            logger.info("S3 client created with assumed role credentials.")
        else:
            if profile:
                session = boto3.Session(profile_name=profile)
                s3_client = session.client("s3", region_name=region)
            else:
                s3_client = boto3.client("s3", region_name=region)
            logger.info("S3 client created with default credentials.")
        return s3_client

    except NoCredentialsError:
        logger.error("Credentials not available.")
        return None
    except Exception as e:
        logger.error(f"An error occurred while creating the S3 client: {e}")
        return None

def upload_part(s3_client, file_path: str, bucket: str, s3_key: str, 
                upload_id: str, part_number: int, start_byte: int, 
                part_size: int, progress_tracker: ProgressTracker, logger) -> dict:
    """マルチパートアップロードの一部をアップロード"""
    try:
        with open(file_path, 'rb') as f:
            f.seek(start_byte)
            data = f.read(part_size)
            
        response = s3_client.upload_part(
            Bucket=bucket,
            Key=s3_key,
            PartNumber=part_number,
            UploadId=upload_id,
            Body=data
        )
        
        # 進捗更新
        progress_tracker.update(len(data))
        
        return {
            'ETag': response['ETag'],
            'PartNumber': part_number
        }
        
    except Exception as e:
        logger.error(f"Failed to upload part {part_number}: {e}")
        raise

def create_transfer_config(options: dict) -> TransferConfig:
    """TransferConfigを作成する"""
    return TransferConfig(
        multipart_threshold=options.get("multipart_threshold", 100 * 1024 * 1024),  # 100MB
        max_concurrency=options.get("max_concurrency", 4),
        multipart_chunksize=options.get("multipart_chunksize", 10 * 1024 * 1024),   # 10MB
        use_threads=options.get("use_threads", True),
        max_io_queue=options.get("max_io_queue", 100),
        io_chunksize=options.get("io_chunksize", 262144),  # 256KB
    )

def upload_file_to_s3(s3_client, file_path: str, bucket_name: str, s3_key: str, 
                            logger, transfer_config: TransferConfig, 
                            enable_progress: bool = True, dry_run: bool = False) -> bool:
    """boto3の高レベルAPIを使用したシンプルなファイルアップロード"""
    
    try:
        if dry_run:
            logger.info(f"[DRY RUN]: Would upload {file_path} to {bucket_name}/{s3_key}")
            return True

        file_size = os.path.getsize(file_path)
        filename = os.path.basename(file_path)
        
        # プログレストラッカー
        progress_tracker = ProgressTracker(file_size, filename) if enable_progress else None
        
        # アップロード実行（boto3が全部やってくれる！）
        s3_client.upload_file(
            file_path, 
            bucket_name, 
            s3_key,
            Config=transfer_config,
            Callback=progress_tracker if progress_tracker else None
        )
        
        if progress_tracker:
            progress_tracker.complete()
        
        logger.info(f"Successfully uploaded {file_path} to {bucket_name}/{s3_key}")
        return True
        
    except FileNotFoundError:
        logger.error(f"File not found: {file_path}")
        return False
    except PermissionError:
        logger.error(f"Permission denied for file: {file_path}")
        return False
    except (NoCredentialsError, ClientError) as e:
        logger.error(f"AWS error uploading {file_path}: {e}")
        return False
    except Exception as e:
        logger.error(f"Unexpected error uploading {file_path}: {e}")
        return False


def upload_file_task(args: tuple) -> Tuple[bool, str]:
    """単一ファイルアップロードタスク（並列処理用・シンプル版）"""
    s3_client, file_path, bucket, s3_key, logger, transfer_config, enable_progress, dry_run = args
    
    try:
        success = upload_file_to_s3(
            s3_client, file_path, bucket, s3_key, 
            logger, transfer_config, enable_progress, dry_run
        )
        return success, file_path
    except Exception as e:
        logger.error(f"Upload task failed for {file_path}: {e}")
        return False, file_path


def process_upload_tasks(s3_client, config, logger):
    """アップロードタスクを処理する（シンプル版）"""

    # 設定取得
    upload_tasks = config.get("upload_tasks", [])
    options = config.get("options", {})
    exclude_patterns = options.get("exclude_patterns", [])
    dry_run = options.get("dry_run", False)
    parallel_uploads = options.get("parallel_uploads", 2)
    enable_progress = options.get("enable_progress", True)

    # TransferConfig作成
    transfer_config = create_transfer_config(options)
    
    # 実行結果の記録
    total_tasks = len(upload_tasks)
    successful_tasks = 0
    failed_tasks = 0

    logger.info(f"Starting upload tasks: {total_tasks} tasks to process (parallel: {parallel_uploads})")
    logger.info(f"Transfer config: threshold={transfer_config.multipart_threshold/1024/1024:.0f}MB, "
                f"concurrency={transfer_config.max_concurrency}, "
                f"chunk={transfer_config.multipart_chunksize/1024/1024:.0f}MB")

    for i, task in enumerate(upload_tasks, 1):
        task_name = task.get("name", f"Task {i}")

        # enabledチェック
        if not task.get("enabled", True):
            logger.info(f"Skipping disabled task: {task_name}")
            continue

        source = task.get("source")
        bucket = task.get("bucket")

        if not source or not bucket:
            logger.error(f"Task {i}/{total_tasks}: {task_name} is missing source or bucket.")
            failed_tasks += 1
            continue

        success = False

        if os.path.isfile(source):
            # 単一ファイルアップロード
            s3_key = task.get("s3_key")
            if not s3_key:
                logger.error(f"Task {i}/{total_tasks}: {task_name} is missing s3_key for file upload.")
                failed_tasks += 1
                continue

            success = upload_file_to_s3(
                s3_client, source, bucket, s3_key, 
                logger, transfer_config, enable_progress, dry_run
            )

        elif os.path.isdir(source):
            # ディレクトリアップロード（並列処理）
            s3_key_prefix = task.get("s3_key_prefix", "")
            recursive = task.get("recursive", False)

            uploaded_count, failed_count = upload_directory(
                s3_client, source, bucket, s3_key_prefix, logger,
                recursive, exclude_patterns, transfer_config, 
                enable_progress, dry_run, parallel_uploads
            )
            success = failed_count == 0
            logger.info(
                f"Task {i}/{total_tasks}: {task_name} - Uploaded {uploaded_count} files, Failed {failed_count} files."
            )

        else:
            logger.error(f"Task {i}/{total_tasks}: {task_name} source is neither a file nor a directory.")
            failed_tasks += 1
            continue

        if success:
            successful_tasks += 1
            logger.info(f"Task {i}/{total_tasks}: {task_name} Success")
        else:
            failed_tasks += 1
            logger.error(f"Task {i}/{total_tasks}: {task_name} Failed")

    logger.info(f"Upload tasks completed: {successful_tasks} successful, {failed_tasks} failed.")
    return successful_tasks, failed_tasks


def upload_directory(s3_client, source: str, bucket: str, s3_key_prefix: str,
                           logger, recursive: bool = False, exclude_patterns: List[str] = None,
                           transfer_config: TransferConfig = None, enable_progress: bool = True,
                           dry_run: bool = False, max_workers: int = 2) -> Tuple[int, int]:
    """ディレクトリ内のファイルを並列でS3にアップロード"""

    if exclude_patterns is None:
        exclude_patterns = []

    # アップロード対象ファイルを収集
    upload_tasks = []
    
    if recursive:
        for root, dirs, files in os.walk(source):
            for file in files:
                file_path = os.path.join(root, file)

                if should_exclude_file(file_path, exclude_patterns):
                    logger.debug(f"Skipping excluded file: {file_path}")
                    continue

                relative_path = os.path.relpath(file_path, source)
                s3_key = s3_key_prefix + relative_path.replace(os.sep, "/")
                
                upload_tasks.append((
                    s3_client, file_path, bucket, s3_key, 
                    logger, transfer_config, enable_progress, dry_run
                ))
    else:
        for item in os.listdir(source):
            file_path = os.path.join(source, item)

            if should_exclude_file(file_path, exclude_patterns):
                logger.debug(f"Skipping excluded file: {file_path}")
                continue

            if os.path.isfile(file_path):
                s3_key = s3_key_prefix + item
                
                upload_tasks.append((
                    s3_client, file_path, bucket, s3_key,
                    logger, transfer_config, enable_progress, dry_run
                ))

    # 並列アップロード実行
    uploaded_count = 0
    failed_count = 0
    
    logger.info(f"Starting parallel upload of {len(upload_tasks)} files with {max_workers} workers")
    
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        # 全てのタスクを投入
        future_to_task = {
            executor.submit(upload_file_task, task): task
            for task in upload_tasks
        }
        
        # 結果を収集
        for future in as_completed(future_to_task):
            try:
                success, file_path = future.result()
                if success:
                    uploaded_count += 1
                else:
                    failed_count += 1
            except Exception as e:
                task = future_to_task[future]
                file_path = task[1]
                logger.error(f"Upload task exception for {file_path}: {e}")
                failed_count += 1

    return uploaded_count, failed_count


def should_exclude_file(file_path, exclude_patterns):
    """ファイルが除外パターンに一致するかチェックする"""
    file_name = os.path.basename(file_path)

    for pattern in exclude_patterns:
        # ファイル名でのマッチ
        if fnmatch.fnmatch(file_name, pattern):
            return True
        if fnmatch.fnmatch(file_path, f"*{pattern}*"):
            return True

    return False


def main():
    # 設定ファイル読み込み
    config = load_config()
    if not config:
        print("Configuration loading failed. Exiting.")
        return

    # ロガーの初期化
    logger = setup_logging(config)

    logger.info("Start uploading file to S3...")

    aws_config = config.get("aws", {})
    if not aws_config:
        logger.error("AWS configuration not found in the config file.")
        return

    # S3クライアント作成
    s3_client = create_s3_client(aws_config, logger)
    if not s3_client:
        logger.error("Failed to create S3 client. Exiting.")
        return

    process_upload_tasks(s3_client, config, logger)


if __name__ == "__main__":
    main()
