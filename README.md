# flynn-pgbackups

[![Build Status](https://travis-ci.org/mattyr/flynn-pgbackups.svg)](https://travis-ci.org/mattyr/flynn-pgbackups)

flynn-pgbackups performs automated backups of all postgres databases in
a [Flynn](https://flynn.io/) cluster to [Amazon
S3](https://aws.amazon.com/s3/).  It is inspired by heroku's pg backups.

## Warning

YMMV, Here be Dragons, etc, etc.  This was whipped up as an internal tool,
but seems others may find it useful.  Pull requests welcome.

## Prerequisites

You already have a flynn cluster set up, and you can operate it using
the flynn command.  You also have an Amazon AWS account and are able to
create/modify S3 buckets and obtain AWS credentials.

## Installation

### 1. Create S3 bucket and create AWS access key

The IAM role only needs permissions to read/write from the chosen s3
bucket.

### 2. Clone this repository

```bash
git clone https://github.com/mattyr/flynn-pgbackups.git
cd flynn-pgbackups
```

### 3. Create a flynn application and provision postgres for the app

flynn-pgbackups uses postgres itself, to track backup histories.

```bash
flynn create pgbackups
flynn resource add postgres
```

### 4. Set flynn app environment variables

- AWS_ACCESS_KEY_ID [required] - from your created AWS credentials
- AWS_SECRET_ACCESS_KEY [required] - from your created AWS credentials
- S3_BUCKET [required] - the bucket to store backups in
- CONTROLLER_KEY [required] - can be obtained with
  ```bash
  flynn -a controller env | grep AUTH_KEY
  ```
- AWS_REGION [optional] - the AWS region for your S3 bucket (defaults to
  "us-east-1")
- SCHEDULE [optional] - backups schedule in cron line format (defaults to
  "0 0 5 \* \* \*", every day at 5AM UTC)
- CONTROLLER_URL [optional] - the internal url for the flynn controller
  (defaults to controller.discoverd) it's unlikely that you'll need to
  change this.
- APPS [optional] - the names of the apps to backup separated by comma. If
  this environment variable is not set, the worker will take backups of
  all flynn applications.

This can be done with a command like:

```bash
flynn env set \
  AWS_ACCESS_KEY_ID=[your-access-key] \
  AWS_SECRET_ACCESS_KEY=[your-secret-key] \
  CONTROLLER_KEY=[your-controller-key] \
  S3_BUCKET=[your-s3-bucket]
```

### 5. Push and scale

```
git push flynn master
flynn scale worker=1
```

## How it works

At the times specified by the SCHEDULE, the worker process takes backups
of the applications specified in the APPS environment variable or all
flynn applications. It obtains a list of all applications using
the Flynn controller API, selecting only those who are using Flynn
postgres (identified by having a "FLYNN_POSTGRES" environment variable).
It then launches a pg_dump job (in a similar fashion to how the flynn
cli command runs "flynn pg dump") and streams the backup to the
configured S3 bucket.  It then cleans up old backups according to the
following rules:

- Keep all backups for the past 7 days
- Keep all Sunday backups for the past 31 days
- Keep all backups for the 1st of every month forever

Which somewhat mimics heroku's backup retention schedule.

## Usage

There's no local CLI yet, but the flynn-pgbackups command supports a few
subcommands that can be run in the cluster to obtain backup information:

- **flynn-bgbackups run**: immediately performs all backups.  Run it
  like this:
  ```bash
  flynn -a pgbackups run flynn-pgbackups run
  ```

- **flynn-pgbackups list [app-name]**: dumps a list of the backups for
  the application specified by app-name.  Run it like this:
  ```bash
  flynn -a pgbackups run flynn-pgbackups list [app-name]
  ```

- **flynn-pgbackups url [backup-id]**: gets a temporary signed url to
  download the backup directly from S3.  Obtain the backup id using the
  "list" command above.  The URL is set to expire in 20 minutes.  Run it
  like this:
  ```bash
  flynn -a pgbackups run flynn-pgbackups url [backup-id]
  ```

## TODO

- Configurable schedules / retention per-app?
- Create a local CLI and API so that running jobs for simple tasks (url,
  run, list, etc) isn't necessary
- Automated restore from S3 backup
- More testing, of course
