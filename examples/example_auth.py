import os
from botocore.auth import SigV4QueryAuth
from botocore.awsrequest import AWSRequest
from botocore.credentials import get_credentials
from botocore.endpoint import URLLib3Session
import boto3
import requests

session = boto3.Session()
credentials = session.get_credentials()
credentials = credentials.get_frozen_credentials()
region = os.getenv("AWS_DEFAULT_REGION", default="us-east-1")
macaroon_url = os.getenv("MACAROON_STORAGE_URL")
signer = SigV4QueryAuth(credentials, "sts", region)

request = AWSRequest(
    method="POST",
    url=f"https://sts.{region}.amazonaws.com",
    params={
        "Action": "GetCallerIdentity",
        "Version": "2011-06-15",
    },
    headers={
      "content-type": "application/x-www-form-urlencoded"
    },
)

signer.add_auth(request)
prep_req = request.prepare()

token = prep_req.url
token = token.replace(f"https://sts.{region}.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15&", "")
print(token)
print()

headers = {
  "X-Amazon-Presigned-Getcalleridentity": token,
}

response = requests.get("{macaroon_storage_url}/get/lnd1", headers=headers)
print(response.text)
