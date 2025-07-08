#!/usr/bin/env python3
"""S3クライアントのテスト"""
from s3_uploader.models.config import Config
from s3_uploader.utils.logger import LoggerManager
from s3_uploader.core.s3_client import S3ClientManager

def test_s3_client():
    """S3クライアントが作成できるか確認"""
    # 設定を読み込み
    config = Config.from_file("config.json")
    
    # ロガーをセットアップ
    logger = LoggerManager.setup(config.logging)
    
    try:
        # S3クライアントを作成
        client_manager = S3ClientManager(config.aws)
        s3_client = client_manager.get_client()
        
        # バケット一覧を取得してテスト
        response = s3_client.list_buckets()
        print("✅ S3クライアントの作成成功！")
        print(f"アクセス可能なバケット数: {len(response['Buckets'])}")
        
        # 設定されているバケットの存在確認
        for task in config.upload_tasks:
            try:
                s3_client.head_bucket(Bucket=task.bucket)
                print(f"✅ バケット '{task.bucket}' にアクセス可能")
            except Exception as e:
                print(f"❌ バケット '{task.bucket}' にアクセスできません: {e}")
                
    except Exception as e:
        print(f"❌ S3クライアントの作成に失敗: {e}")

if __name__ == "__main__":
    test_s3_client()