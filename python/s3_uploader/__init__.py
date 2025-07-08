"""S3 Uploader パッケージ"""
from typing import Tuple
from .models.config import Config
from .utils.logger import LoggerManager
from .core.task_runner import TaskRunner


class S3Uploader:
    """S3アップローダーのメインクラス"""
    
    def __init__(self, config_path: str = "config.json"):
        # 設定を読み込み
        self.config = Config.from_file(config_path)
        
        # ロガーをセットアップ
        self.logger = LoggerManager.setup(self.config.logging)
        self.logger.info("S3 Uploader initialized")
        
        # タスクランナーを作成
        self.task_runner = TaskRunner(self.config)
        
    def run(self) -> Tuple[int, int]:
        """アップロードタスクを実行"""
        self.logger.info("Starting S3 upload process...")
        return self.task_runner.run_all_tasks()


__all__ = ['S3Uploader', 'Config']