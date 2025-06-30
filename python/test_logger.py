#!/usr/bin/env python3
"""ロガーのテスト"""
from s3_uploader.models.config import Config
from s3_uploader.utils.logger import LoggerManager

def test_logger():
    """ロガーが正しく動作するか確認"""
    config = Config.from_file("config.json")
    logger = LoggerManager.setup(config.logging)
    
    logger.info("✅ ロガーのセットアップ成功！")
    logger.debug("デバッグメッセージ（INFOレベルでは表示されないはず）")
    logger.warning("⚠️ 警告メッセージ")
    logger.error("❌ エラーメッセージ")
    
    print("\nログファイルも確認してみて: logs/s3_uploader.log")

if __name__ == "__main__":
    test_logger()