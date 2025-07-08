#!/usr/bin/env python3
"""設定クラスのテスト"""
from s3_uploader.models.config import Config

def test_config_loading():
    """既存のconfig.jsonが読み込めるか確認"""
    try:
        config = Config.from_file("config.json")
        print("✅ 設定ファイルの読み込み成功！")
        print(f"  - AWS Region: {config.aws.region}")
        print(f"  - Log Level: {config.logging.level}")
        print(f"  - Upload Tasks: {len(config.upload_tasks)}")
        
        for i, task in enumerate(config.upload_tasks):
            print(f"  - Task {i+1}: {task.name} ({task.source} -> {task.bucket})")
            
    except Exception as e:
        print(f"❌ エラー: {e}")

if __name__ == "__main__":
    test_config_loading()