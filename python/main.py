import boto3
import json
from botocore.exceptions import NoCredentialsError

s3_client = boto3.client('s3')

def load_config(config_path="config.json"):
    """設定ファイルを読み込む"""
    try:
        with open(config_path, 'r', encoding='utf-8') as file:
            config = json.load(file)
        return config
    except FileNotFoundError:
        print(f"Configuration file {config_path} not found.")
        return None
    except json.JSONDecodeError as e:
        print(f"Error decoding JSON from the configuration file {config_path}: {e}")
        return None
    except Exception as e:
        print(f"An error occurred while loading the configuration: {e}")
        return None

def create_s3_client(aws_config):
    """S3クライアントを作成する"""
    try:
        region = aws_config.get('region')
        profile = aws_config.get('profile')

        if profile:
            session = boto3.Session(profile_name=profile)
            s3_client = session.client('s3', region_name=region)
        else:
            s3_client = boto3.client('s3', region_name=region)
        return s3_client
    
    except NoCredentialsError:
        print("Credentials not available.")
        return None
    except Exception as e:
        print(f"An error occurred while creating the S3 client: {e}")
        return None

def upload_file_to_s3(s3_client, file_path, bucket_name, s3_key):
    """ファイルをS3にアップロード"""
    try:
        s3_client.upload_file(file_path, bucket_name, s3_key)
        print(f"File {file_path} uploaded to {bucket_name}/{s3_key}")
        return True
    except FileNotFoundError:
        print(f"The file {file_path} was not found.")
        return False
    except NoCredentialsError:
        print("Credentials not available.")
        return False
    except Exception as e:
        print(f"An error occurred: {e}")
        return False

def main():
    print("Start uploading file to S3...")

    #設定ファイル読み込み
    config = load_config()
    if not config:
        print("Configuration loading failed. Exiting.")
        return
    
    aws_config = config.get('aws', {})
    if not aws_config:
        print("AWS configuration not found in the config file.")
        return
    
    # S3クライアント作成
    s3_client = create_s3_client(aws_config)
    if not s3_client:
        print("Failed to create S3 client. Exiting.")
        return

    file_path = "../test-data/sample_data.csv"
    bucket_name = "s3-experiment-bucket-250615"
    s3_key = 'sample_data.csv'

    # ファイルをS3にアップロード
    upload_status = upload_file_to_s3(s3_client, file_path, bucket_name, s3_key)

    if upload_status:
        print("File upload was successful.")
    else:
        print("File upload failed.")

if __name__ == "__main__":
    main()
