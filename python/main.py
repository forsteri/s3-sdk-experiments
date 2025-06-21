import boto3
from botocore.exceptions import NoCredentialsError

s3_client = boto3.client('s3')

def upload_file_to_s3(file_path, bucket_name, s3_key):
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
    print("Hello from python!")

    upload_status = upload_file_to_s3(
        '../test-data/sample_data.csv', 
        's3-experiment-bucket-250615', 
        'sample_data.csv'
    )
    
    if upload_status:
        print("File upload was successful.")
    else:
        print("File upload failed.")

if __name__ == "__main__":
    main()
