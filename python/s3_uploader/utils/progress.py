"""アップロード進捗管理"""
import time
import threading
from typing import Dict, Optional


class ProgressTracker:
    """単一ファイルのアップロード進捗を追跡"""
    
    def __init__(self, total_size: int, filename: str):
        self.total_size = total_size
        self.filename = filename
        self.uploaded_size = 0
        self.lock = threading.Lock()
        self.start_time = time.time()
        
    def __call__(self, bytes_transferred: int):
        """boto3のコールバック関数として使用"""
        with self.lock:
            self.uploaded_size += bytes_transferred
            self._display_progress()
            
    def _display_progress(self):
        """進捗を表示"""
        if self.total_size == 0:
            return
            
        progress = (self.uploaded_size / self.total_size) * 100
        elapsed_time = time.time() - self.start_time
        
        if elapsed_time > 0:
            speed = self.uploaded_size / elapsed_time / 1024 / 1024  # MB/s
            eta = (self.total_size - self.uploaded_size) / (self.uploaded_size / elapsed_time) if self.uploaded_size > 0 else 0
            
            print(f"\r{self.filename}: {progress:.1f}% ({self.uploaded_size}/{self.total_size}) "
                  f"- {speed:.2f} MB/s - ETA: {eta:.0f}s", end="", flush=True)
    
    def complete(self):
        """アップロード完了"""
        elapsed_time = time.time() - self.start_time
        speed = self.total_size / elapsed_time / 1024 / 1024 if elapsed_time > 0 else 0
        print(f"\r{self.filename}: Complete! - {speed:.2f} MB/s - {elapsed_time:.1f}s")


class ProgressManager:
    """複数のアップロードタスクの進捗を管理"""
    
    def __init__(self):
        self.trackers: Dict[str, ProgressTracker] = {}
        self.lock = threading.Lock()
        
    def create_tracker(self, task_id: str, total_size: int, filename: str) -> ProgressTracker:
        """新しいトラッカーを作成"""
        with self.lock:
            tracker = ProgressTracker(total_size, filename)
            self.trackers[task_id] = tracker
            return tracker
    
    def get_tracker(self, task_id: str) -> Optional[ProgressTracker]:
        """既存のトラッカーを取得"""
        return self.trackers.get(task_id)
    
    def remove_tracker(self, task_id: str):
        """完了したトラッカーを削除"""
        with self.lock:
            self.trackers.pop(task_id, None)