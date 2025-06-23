import boto3
import json
import os
import logging
import fnmatch
import time
from botocore.exceptions import NoCredentialsError, ClientError

def setup_logging(config):
    """ロギングの設定を初期化する"""
    logging_config = config.get('logging', {})

    # デフォルト値
    level = logging_config.get('level', 'INFO')
    format_str = logging_config.get('format', '%(asctime)s - %(levelname)s - %(message)s')
    log_file = logging_config.get('file', None)

    # ログレベルの設定
    log_level = getattr(logging, level.upper(), logging.INFO)

    # ロガーの設定
    handlers = []

    # コンソール出力
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(logging.Formatter(format_str, datefmt='%Y-%m-%d %H:%M:%S'))
    handlers.append(console_handler)

    if log_file:
        log_dir = os.path.dirname(log_file)
        if log_dir and not os.path.exists(log_dir):
            os.makedirs(log_dir)

        file_handler = logging.FileHandler(log_file, encoding='utf-8')
        file_handler.setFormatter(logging.Formatter(format_str, datefmt='%Y-%m-%d %H:%M:%S'))
        handlers.append(file_handler)

    # ロガーの設定
    logging.basicConfig(
        level=log_level,
        handlers=handlers
    )

    return logging.getLogger(__name__)


def load_config(config_path="config.json"):
    """設定ファイルを読み込む"""
    try:
        with open(config_path, 'r', encoding='utf-8') as file:
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
    assume_role_config = aws_config.get('assume_role')
    if not assume_role_config:
        logger.error("Assume role configuration not found.")
        return None

    try:
        region = aws_config.get('region')
        endpoint_url = f"https://sts.{region}.amazonaws.com"
        profile = aws_config.get('profile')

        if profile:
            session = boto3.Session(profile_name=profile)
            sts_client = session.client('sts', region_name=region, endpoint_url=endpoint_url)
        else:
            sts_client = boto3.client('sts', region_name=region, endpoint_url=endpoint_url)

        # Assume Roleを実行
        role_arn = assume_role_config['role_arn']
        session_name = assume_role_config['session_name']
        external_id = assume_role_config.get('external_id')
        duration = assume_role_config.get('duration_seconds', 3600)

        assume_role_params = {
            'RoleArn': role_arn,
            'RoleSessionName': session_name,
            'DurationSeconds': duration
        }

        if external_id:
            assume_role_params['ExternalId'] = external_id

        response = sts_client.assume_role(**assume_role_params)
        credentials = response['Credentials']
        logger.info(f"Assumed role successfully: {role_arn}")
        return {
            'access_key_id': credentials['AccessKeyId'],
            'secret_access_key': credentials['SecretAccessKey'],
            'session_token': credentials['SessionToken'],
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
        region = aws_config.get('region')
        profile = aws_config.get('profile')

        temp_credentials = assume_role(aws_config, logger)
        if temp_credentials:
            s3_client = boto3.client(
                's3',
                region_name=region,
                aws_access_key_id=temp_credentials['access_key_id'],
                aws_secret_access_key=temp_credentials['secret_access_key'],
                aws_session_token=temp_credentials['session_token']
            )
            logger.info("S3 client created with assumed role credentials.")
        else:
            if profile:
                session = boto3.Session(profile_name=profile)
                s3_client = session.client('s3', region_name=region)
            else:
                s3_client = boto3.client('s3', region_name=region)
            logger.info("S3 client created with default credentials.")
        return s3_client
    
    except NoCredentialsError:
        logger.error("Credentials not available.")
        return None
    except Exception as e:
        logger.error(f"An error occurred while creating the S3 client: {e}")
        return None

def upload_file_to_s3(s3_client, file_path, bucket_name, s3_key, logger, dry_run=False, max_retries=3):
    """ファイルをS3にアップロード"""
    
    for attempt in range(max_retries + 1):
        try:
            if dry_run:
                logger.info(f"[DRY RUN]: Would upload {file_path} to {bucket_name}/{s3_key}")
                return True

            s3_client.upload_file(file_path, bucket_name, s3_key)
            logger.info(f"File {file_path} uploaded to {bucket_name}/{s3_key}")
            return True

        except FileNotFoundError:
            logger.error(f"File not found: {file_path}")
            return False
        except PermissionError:
            logger.error(f"Permission denied for file: {file_path}")
            return False
        except (NoCredentialsError, ClientError, ConnectionError, TimeoutError) as e:
            if attempt < max_retries:
                wait_time = 2 ** attempt  # 1, 2, 4秒
                logger.warning(f"Upload failed (attempt {attempt + 1}/{max_retries + 1}): {e}")
                logger.info(f"Retrying in {wait_time} seconds...")
                time.sleep(wait_time)
            else:
                logger.error(f"Upload failed after {max_retries + 1} attempts: {e}")
                return False
        except Exception as e:
            logger.error(f"An unexpected error occurred while uploading {file_path}: {e}")
            return False

def process_upload_tasks(s3_client, config, logger):
    """アップロードタスクを処理する"""
    
    # 設定取得
    upload_tasks = config.get('upload_tasks', [])
    options = config.get('options', {})
    exclude_patterns = options.get('exclude_patterns', [])
    dry_run = options.get('dry_run', False)
    max_retries = options.get('max_retries', 3)

    # 実行結果の記録
    total_tasks = len(upload_tasks)
    successful_tasks = 0
    failed_tasks = 0

    logger.info(f"Starting upload tasks: {total_tasks} tasks to process.")

    for i, task in enumerate(upload_tasks, 1):
        task_name = task.get('name', f"Task {i}")

        # enabledチェック
        if not task.get('enabled', True):
            logger.info(f"Skipping disabled task: {task_name}")
            continue

        source = task.get('source')
        bucket = task.get('bucket')

        if not source or not bucket:
            logger.error(f"Task {i}/{total_tasks}: {task_name} is missing source or bucket.")
            failed_tasks += 1
            continue

        success = False

        if os.path.isfile(source):
            s3_key = task.get('s3_key')
            if not s3_key:
                logger.error(f"Task {i}/{total_tasks}: {task_name} is missing s3_key for file upload.")
                failed_tasks += 1
                continue

            success = upload_file_to_s3(s3_client, source, bucket, s3_key, logger, dry_run, max_retries)

        elif os.path.isdir(source):
            s3_key_prefix = task.get('s3_key_prefix', '')
            recursive = task.get('recursive', False)

            uploaded_count, failed_count = upload_directory(
                s3_client, source, bucket, s3_key_prefix, logger, recursive, exclude_patterns, dry_run, max_retries
            )
            success = (failed_count == 0)
            logger.info(f"Task {i}/{total_tasks}: {task_name} - Uploaded {uploaded_count} files, Failed {failed_count} files.")

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

def upload_directory(s3_client, source, bucket, s3_key_prefix, logger, recursive=False, exclude_patterns=None, dry_run=False, max_retries=3):
    """ディレクトリ内のファイルをS3にアップロード"""

    if exclude_patterns is None:
        exclude_patterns = []

    uploaded_count = 0
    failed_count = 0

    if recursive:
        for root, dirs, files in os.walk(source):
            for file in files:
                file_path = os.path.join(root, file)

                if should_exclude_file(file_path, exclude_patterns):
                    logger.info(f"Skipping excluded file: {file_path}")
                    continue

                relative_path = os.path.relpath(file_path, source)
                s3_key = s3_key_prefix + relative_path.replace(os.sep, '/')

                if upload_file_to_s3(s3_client, file_path, bucket, s3_key, logger):
                    uploaded_count += 1
                else:
                    failed_count += 1

    else:
        for item in os.listdir(source):
            file_path = os.path.join(source, item)
    
            if should_exclude_file(file_path, exclude_patterns):
                logger.debug(f"Skipping excluded file: {file_path}")
                continue

            if os.path.isfile(file_path):
                s3_key = s3_key_prefix + item

                if upload_file_to_s3(s3_client, file_path, bucket, s3_key, logger, dry_run, max_retries):
                    uploaded_count += 1
                else:
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
    #設定ファイル読み込み
    config = load_config()
    if not config:
        print("Configuration loading failed. Exiting.")
        return

    # ロガーの初期化
    logger = setup_logging(config)

    logger.info("Start uploading file to S3...")

    aws_config = config.get('aws', {})
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
