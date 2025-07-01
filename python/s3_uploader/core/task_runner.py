"""アップロードタスクの実行"""
import os
from typing import List, Tuple

from ..models.config import UploadTask, Config
from ..utils.logger import LoggerManager
from ..utils.file_utils import FileScanner
from .uploader import UploadExecutor, ParallelUploadExecutor
from .s3_client import S3ClientManager


class TaskRunner:
    """アップロードタスクを実行"""
    
    def __init__(self, config: Config):
        self.config = config
        self.logger = LoggerManager.get_logger()
        
        # S3クライアントとアップローダーを初期化
        client_manager = S3ClientManager(config.aws)
        self.s3_client = client_manager.get_client()
        self.executor = UploadExecutor(self.s3_client, config.options)
        self.parallel_executor = ParallelUploadExecutor(
            self.executor, 
            config.options.parallel_uploads
        )
        self.file_scanner = FileScanner(config.options.exclude_patterns)
        
    def run_all_tasks(self) -> Tuple[int, int]:
        """全てのタスクを実行"""
        total_tasks = len(self.config.upload_tasks)
        successful_tasks = 0
        failed_tasks = 0
        
        self.logger.info(f"Starting upload tasks: {total_tasks} tasks to process")
        
        for i, task in enumerate(self.config.upload_tasks, 1):
            if not task.enabled:
                self.logger.info(f"Skipping disabled task: {task.name}")
                continue
                
            self.logger.info(f"Task {i}/{total_tasks}: Starting '{task.name}'")
            
            try:
                success = self._run_single_task(task)
                if success:
                    successful_tasks += 1
                    self.logger.info(f"Task {i}/{total_tasks}: '{task.name}' completed successfully")
                else:
                    failed_tasks += 1
                    self.logger.error(f"Task {i}/{total_tasks}: '{task.name}' failed")
            except Exception as e:
                failed_tasks += 1
                self.logger.error(f"Task {i}/{total_tasks}: '{task.name}' failed with error: {e}")
                
        self.logger.info(
            f"Upload tasks completed: {successful_tasks} successful, {failed_tasks} failed"
        )
        return successful_tasks, failed_tasks
        
    def _run_single_task(self, task: UploadTask) -> bool:
        """単一タスクを実行"""
        if os.path.isfile(task.source):
            # 単一ファイルのアップロード
            return self._upload_single_file(task)
        elif os.path.isdir(task.source):
            # ディレクトリのアップロード
            return self._upload_directory(task)
        else:
            self.logger.error(f"Source is neither file nor directory: {task.source}")
            return False
            
    def _upload_single_file(self, task: UploadTask) -> bool:
        """単一ファイルをアップロード"""
        if not task.s3_key:
            self.logger.error(f"s3_key is required for file upload: {task.name}")
            return False
            
        try:
            file_info = self.file_scanner.get_file_info(task.source)
            result = self.executor.upload_file(file_info, task.bucket, task.s3_key)
            return result.success
        except Exception as e:
            self.logger.error(f"Error uploading file {task.source}: {e}")
            return False
            
    def _upload_directory(self, task: UploadTask) -> bool:
        """ディレクトリをアップロード"""
        s3_key_prefix = task.s3_key_prefix or ""
        
        try:
            # アップロードタスクを収集
            upload_tasks = []
            for file_info in self.file_scanner.scan_directory(task.source, task.recursive):
                s3_key = s3_key_prefix + file_info.relative_path.replace(os.sep, "/")
                upload_tasks.append((file_info, task.bucket, s3_key))
                
            if not upload_tasks:
                self.logger.warning(f"No files found in {task.source}")
                return True
                
            # 並列アップロード実行
            successful, failed = self.parallel_executor.upload_files(upload_tasks)
            
            self.logger.info(
                f"Directory upload completed: {successful} successful, {failed} failed"
            )
            return failed == 0
            
        except Exception as e:
            self.logger.error(f"Error uploading directory {task.source}: {e}")
            return False