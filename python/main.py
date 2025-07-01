#!/usr/bin/env python3
"""S3 Uploader - 新しいエントリーポイント"""
from s3_uploader import S3Uploader


def main():
    """メイン関数"""
    try:
        uploader = S3Uploader("config.json")
        successful, failed = uploader.run()
        
        # 終了コードを設定
        exit_code = 0 if failed == 0 else 1
        exit(exit_code)
        
    except Exception as e:
        print(f"Error: {e}")
        exit(1)


if __name__ == "__main__":
    main()