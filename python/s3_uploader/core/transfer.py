"""S3転送設定管理"""
from boto3.s3.transfer import TransferConfig as BotoTransferConfig
from ..models.config import UploadOptions


class TransferConfigManager:
    """S3転送設定の管理"""
    
    @staticmethod
    def create_config(options: UploadOptions) -> BotoTransferConfig:
        """UploadOptionsからTransferConfigを作成"""
        return BotoTransferConfig(
            multipart_threshold=options.multipart_threshold,
            max_concurrency=options.max_concurrency,
            multipart_chunksize=options.multipart_chunksize,
            use_threads=options.use_threads,
            max_io_queue=options.max_io_queue,
            io_chunksize=options.io_chunksize,
        )