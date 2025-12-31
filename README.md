# /tri/

A fast and simple S3-compatible server. It stores the data on local fs and uses extended attributes for metadata - xattr are supported by most file systems on Linux and MacOS and works with NFSv4.2 and later.

**IMPORTANT:** This is not production-ready software. This project is in active development.

Supports Authorization Header (AWS Signature Version 4)

Client side tested with:
* aws-cli/2.13.30 or greater
* aws-sdk-go-v2 v1.22.1
* aws-sdk-ruby3/3.185.2

## Usage

Create aws config and credentials files

cat $HOME/.aws/config

```shell
[default]
endpoint_url=http://localhost:3000
```

cat $HOME/.aws/credentials

```shell
[default]
aws_access_key_id = user
aws_secret_access_key = password
region = us-east-1
```

Run tri server

```shell
go run main.go
```

Create bucket

```shell
aws s3 mb s3://test-bucket
```

List buckets

```shell
aws s3 ls

2025-12-30 17:15:01 test-bucket
```
