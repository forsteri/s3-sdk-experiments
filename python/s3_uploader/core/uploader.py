"""S3アップロード実行クラス"""
import os
from typing import Optional, Tuple, List
from dataclasses import dataclass
from concurrent.futures import ThreadPoolExecutor, as_completed
import time

from botocore.exceptions import ClientError, NoCredentialsError
from ..models.config import UploadTask, UploadOptions
from ..utils.logger import LoggerManager
from ..utils.progress import ProgressTracker, ProgressManager
from ..utils.file_utils import FileScanner, FileInfo
from .transfer import TransferConfigManager


@dataclass
class UploadResult:
    """アップロード結果"""
    file_path: str
    success: bool
    error: Optional[str] = None
    
    
class UploadExecutor:
    """ファイルアップロードの実行"""
    
    def __init__(self, s3_client, options: UploadOptions):
        self.s3_client = s3_client
        self.options = options
        self.logger = LoggerManager.get_logger()
        self.transfer_config = TransferConfigManager.create_config(options)
        self.progress_manager = ProgressManager()
        
    def upload_file(self, file_info: FileInfo, bucket: str, s3_key: str) -> UploadResult:
        """単一ファイルをアップロード"""
        if self.options.dry_run:
            self.logger.info(f"[DRY RUN]: Would upload {file_info.path} to {bucket}/{s3_key}")
            return UploadResult(file_info.path, success=True)
            
        # リトライ処理
        for attempt in range(self.options.max_retries + 1):
            try:
                return self._execute_upload(file_info, bucket, s3_key)
            except Exception as e:
                if attempt < self.options.max_retries:
                    wait_time = 2 ** attempt  # 指数バックオフ
                    self.logger.warning(
                        f"Upload failed (attempt {attempt + 1}/{self.options.max_retries + 1}), "
                        f"retrying in {wait_time}s: {file_info.path}"
                    )
                    time.sleep(wait_time)
                else:
                    self.logger.error(
                        f"Upload failed after {self.options.max_retries + 1} attempts: {file_info.path}"
                    )
                    return UploadResult(file_info.path, success=False, error=str(e))
                    
    def _execute_upload(self, file_info: FileInfo, bucket: str, s3_key: str) -> UploadResult:
        """実際のアップロード処理"""
        try:
            # プログレストラッカー
            progress_tracker = None
            if self.options.enable_progress:
                progress_tracker = self.progress_manager.create_tracker(
                    file_info.path, file_info.size, file_info.name
                )
            
            # アップロード実行
            self.s3_client.upload_file(
                file_info.path,
                bucket,
                s3_key,
                Config=self.transfer_config,
                Callback=progress_tracker if progress_tracker else None
            )
            
            if progress_tracker:
                progress_tracker.complete()
                self.progress_manager.remove_tracker(file_info.path)
                
            self.logger.info(f"Successfully uploaded {file_info.path} to {bucket}/{s3_key}")
            return UploadResult(file_info.path, success=True)
            
        except FileNotFoundError:
            error = f"File not found: {file_info.path}"
            self.logger.error(error)
            return UploadResult(file_info.path, success=False, error=error)
        except PermissionError:
            error = f"Permission denied for file: {file_info.path}"
            self.logger.error(error)
            return UploadResult(file_info.path, success=False, error=error)
        except (NoCredentialsError, ClientError) as e:
            error = f"AWS error uploading {file_info.path}: {e}"
            self.logger.error(error)
            return UploadResult(file_info.path, success=False, error=str(e))
        except Exception as e:
            error = f"Unexpected error uploading {file_info.path}: {e}"
            self.logger.error(error)
            return UploadResult(file_info.path, success=False, error=str(e))


class ParallelUploadExecutor:
    """並列アップロード実行"""
    
    def __init__(self, executor: UploadExecutor, max_workers: int = 2):
        self.executor = executor
        self.max_workers = max_workers
        self.logger = LoggerManager.get_logger()
        
    def upload_files(self, upload_tasks: List[Tuple[FileInfo, str, str]]) -> Tuple[int, int]:
        """複数ファイルを並列でアップロード
        
        Args:
            upload_tasks: (FileInfo, bucket, s3_key) のタプルのリスト
            
        Returns:
            (成功数, 失敗数) のタプル
        """
        total_files = len(upload_tasks)
        self.logger.info(
            f"Starting parallel upload of {total_files} files with {self.max_workers} workers"
        )
        
        successful = 0
        failed = 0
        
        with ThreadPoolExecutor(max_workers=self.max_workers) as pool:
            # タスクを投入
            future_to_task = {
                pool.submit(self.executor.upload_file, file_info, bucket, s3_key): (file_info, bucket, s3_key)
                for file_info, bucket, s3_key in upload_tasks
            }
            
            # 結果を収集
            for future in as_completed(future_to_task):
                file_info, _, _ = future_to_task[future]
                try:
                    result = future.result()
                    if result.success:
                        successful += 1
                    else:
                        failed += 1
                except Exception as e:
                    self.logger.error(f"Upload task exception for {file_info.path}: {e}")
                    failed += 1
                    
        return successful, failed