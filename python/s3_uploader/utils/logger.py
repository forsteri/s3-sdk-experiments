"""ロギング設定ユーティリティ"""
import logging
import os
from typing import Optional, List
from ..models.config import LoggingConfig


class LoggerManager:
    """ロガーの設定と管理"""
    
    _logger: Optional[logging.Logger] = None
    
    @classmethod
    def setup(cls, config: LoggingConfig) -> logging.Logger:
        """ロガーをセットアップ"""
        if cls._logger is not None:
            return cls._logger
            
        # ログレベルの設定
        log_level = getattr(logging, config.level.upper(), logging.INFO)
        
        # ハンドラーの準備
        handlers: List[logging.Handler] = []
        
        # フォーマッターの作成
        formatter = logging.Formatter(
            config.format,
            datefmt="%Y-%m-%d %H:%M:%S"
        )
        
        # コンソールハンドラー
        console_handler = logging.StreamHandler()
        console_handler.setFormatter(formatter)
        handlers.append(console_handler)
        
        # ファイルハンドラー（設定されている場合）
        if config.file:
            log_dir = os.path.dirname(config.file)
            if log_dir and not os.path.exists(log_dir):
                os.makedirs(log_dir)
                
            file_handler = logging.FileHandler(config.file, encoding="utf-8")
            file_handler.setFormatter(formatter)
            handlers.append(file_handler)
        
        # ロガーの設定
        logger = logging.getLogger("s3_uploader")
        logger.setLevel(log_level)
        logger.handlers = handlers
        
        cls._logger = logger
        return logger
    
    @classmethod
    def get_logger(cls) -> logging.Logger:
        """設定済みのロガーを取得"""
        if cls._logger is None:
            raise RuntimeError("Logger not initialized. Call setup() first.")
        return cls._logger