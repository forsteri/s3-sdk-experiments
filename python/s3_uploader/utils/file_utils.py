"""ファイル操作関連のユーティリティ"""
import os
import fnmatch
from typing import List, Tuple, Generator
from dataclasses import dataclass


@dataclass
class FileInfo:
    """ファイル情報"""
    path: str
    size: int
    relative_path: str
    
    @property
    def name(self) -> str:
        return os.path.basename(self.path)


class FileScanner:
    """ファイルスキャン機能"""
    
    def __init__(self, exclude_patterns: List[str] = None):
        self.exclude_patterns = exclude_patterns or []
        
    def should_exclude(self, file_path: str) -> bool:
        """ファイルが除外パターンに一致するかチェック"""
        file_name = os.path.basename(file_path)
        
        for pattern in self.exclude_patterns:
            # ファイル名でのマッチ
            if fnmatch.fnmatch(file_name, pattern):
                return True
            # パス全体でのマッチ
            if fnmatch.fnmatch(file_path, f"*{pattern}*"):
                return True
                
        return False
        
    def scan_directory(self, directory: str, recursive: bool = False) -> Generator[FileInfo, None, None]:
        """ディレクトリをスキャンしてファイル情報を生成"""
        if not os.path.isdir(directory):
            raise ValueError(f"Not a directory: {directory}")
            
        if recursive:
            for root, dirs, files in os.walk(directory):
                # 除外パターンに一致するディレクトリをスキップ
                dirs[:] = [d for d in dirs if not self.should_exclude(os.path.join(root, d))]
                
                for file in files:
                    file_path = os.path.join(root, file)
                    if not self.should_exclude(file_path):
                        relative_path = os.path.relpath(file_path, directory)
                        yield FileInfo(
                            path=file_path,
                            size=os.path.getsize(file_path),
                            relative_path=relative_path
                        )
        else:
            for item in os.listdir(directory):
                file_path = os.path.join(directory, item)
                if os.path.isfile(file_path) and not self.should_exclude(file_path):
                    yield FileInfo(
                        path=file_path,
                        size=os.path.getsize(file_path),
                        relative_path=item
                    )
    
    def get_file_info(self, file_path: str) -> FileInfo:
        """単一ファイルの情報を取得"""
        if not os.path.isfile(file_path):
            raise ValueError(f"Not a file: {file_path}")
            
        return FileInfo(
            path=file_path,
            size=os.path.getsize(file_path),
            relative_path=os.path.basename(file_path)
        )