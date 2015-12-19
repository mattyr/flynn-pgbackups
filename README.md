# flynn-pgbackups

[![Build Status](https://travis-ci.org/mattyr/flynn-pgbackups.svg)](https://travis-ci.org/mattyr/flynn-pgbackups)

flynn-pgbackups performs automated backups of all postgres databases in
a [Flynn](https://flynn.io/) cluster to [Amazon
S3](https://aws.amazon.com/s3/).  It is inspired by heroku's pg backups.

## Prerequisites

You already have a flynn cluster set up, and you can operate it using
the flynn command.  You also have an Amazon AWS account and are able to
create/modify S3 buckets and obtain AWS credentials.

## Installation

### 1. Clone this repository

```bash
git clone https://github.com/mattyr/flynn-pgbackups.git
cd flynn-pgbackups
```

### 2. Create S3 bucket and create AWS access key

The IAM role only needs permissions to read/write from the chosen s3
bucket.

### 3. Create a flynn application

```bash
flynn create pgbackups
```

### 4. Set flynn app environment variables

- AWS_ACCESS_KEY_ID [required] - from your created AWS credentials
- AWS_SECRET_ACCESS_KEY [required] - from your created AWS credentials
- CONTROLLER_KEY [required] - can be obtained with
  ```bash
  flynn -a controller env | grep AUTH_KEY
  ```
- AWS_REGION [optional] - the AWS region for your S3 bucket (defaults to
  "us-east-1")
- 

