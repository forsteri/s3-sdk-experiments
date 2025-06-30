"""S3クライアント管理"""
import boto3
from typing import Optional, Dict, Any
from botocore.exceptions import NoCredentialsError, ClientError
from ..models.config import AWSConfig, AssumeRoleConfig
from ..utils.logger import LoggerManager


class S3ClientManager:
    """S3クライアントの作成と管理"""
    
    def __init__(self, aws_config: AWSConfig):
        self.aws_config = aws_config
        self.logger = LoggerManager.get_logger()
        self._client: Optional[boto3.client] = None
        
    def get_client(self) -> boto3.client:
        """S3クライアントを取得（必要に応じて作成）"""
        if self._client is None:
            self._client = self._create_client()
        return self._client
        
    def _create_client(self) -> boto3.client:
        """S3クライアントを作成"""
        try:
            if self.aws_config.assume_role:
                # AssumeRoleを使用
                temp_credentials = self._assume_role()
                if temp_credentials:
                    s3_client = boto3.client(
                        's3',
                        region_name=self.aws_config.region,
                        aws_access_key_id=temp_credentials['access_key_id'],
                        aws_secret_access_key=temp_credentials['secret_access_key'],
                        aws_session_token=temp_credentials['session_token']
                    )
                    self.logger.info("S3 client created with assumed role credentials.")
                    return s3_client
                    
            # 通常の認証
            if self.aws_config.profile:
                session = boto3.Session(profile_name=self.aws_config.profile)
                s3_client = session.client('s3', region_name=self.aws_config.region)
            else:
                s3_client = boto3.client('s3', region_name=self.aws_config.region)
                
            self.logger.info("S3 client created with default credentials.")
            return s3_client
            
        except NoCredentialsError:
            self.logger.error("AWS credentials not available.")
            raise
        except Exception as e:
            self.logger.error(f"Error creating S3 client: {e}")
            raise
            
    def _assume_role(self) -> Optional[Dict[str, str]]:
        """AssumeRoleを実行して一時的な認証情報を取得"""
        assume_role_config = self.aws_config.assume_role
        
        try:
            # STSクライアントを作成
            endpoint_url = f"https://sts.{self.aws_config.region}.amazonaws.com"
            
            if self.aws_config.profile:
                session = boto3.Session(profile_name=self.aws_config.profile)
                sts_client = session.client(
                    'sts', 
                    region_name=self.aws_config.region,
                    endpoint_url=endpoint_url
                )
            else:
                sts_client = boto3.client(
                    'sts',
                    region_name=self.aws_config.region,
                    endpoint_url=endpoint_url
                )
                
            # AssumeRoleを実行
            assume_role_params = {
                'RoleArn': assume_role_config.role_arn,
                'RoleSessionName': assume_role_config.session_name,
                'DurationSeconds': assume_role_config.duration_seconds,
            }
            
            if assume_role_config.external_id:
                assume_role_params['ExternalId'] = assume_role_config.external_id
                
            response = sts_client.assume_role(**assume_role_params)
            credentials = response['Credentials']
            
            self.logger.info(f"Assumed role successfully: {assume_role_config.role_arn}")
            
            return {
                'access_key_id': credentials['AccessKeyId'],
                'secret_access_key': credentials['SecretAccessKey'],
                'session_token': credentials['SessionToken']
            }
            
        except ClientError as e:
            self.logger.error(f"Error assuming role: {e}")
            return None
        except Exception as e:
            self.logger.error(f"Unexpected error during assume role: {e}")
            return None