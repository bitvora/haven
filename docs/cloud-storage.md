# Cloud Storage Provider Specific Instructions

This page contains specific configuration examples for various cloud storage providers. For general instructions on how to set up periodic cloud backups, see the [Backup and Restore Documentation](backup.md#periodic-cloud-backups).

## Provider Specific Instructions

### AWS S3

For AWS S3, set the appropriate endpoint for your region/availability zone:

```Dotenv
S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
S3_ENDPOINT="s3.us-east-1.amazonaws.com"
S3_REGION="us-east-1"
S3_BUCKET_NAME="haven_backup"
```

### GCP Cloud Storage

For GCP, you can set `S3_ENDPOINT` to `storage.googleapis.com`. 

`S3_REGION` can be left blank. `S3_ACCESS_KEY_ID` and `S3_SECRET_KEY` needs to be set to a [HMAC key](
https://cloud.google.com/storage/docs/authentication/hmackeys), see GCP's official documentation on [how to create a HMAC 
key for a service account](https://cloud.google.com/storage/docs/authentication/managing-hmackeys#create).

```Dotenv
S3_ACCESS_KEY_ID="GOOGXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
S3_SECRET_KEY="Yyy+YYY0/yYYYYyyyy0+YyyYyyYyyYyyyyYyyYyy"
S3_ENDPOINT="storage.googleapis.com"
S3_REGION=""
S3_BUCKET_NAME="haven_backup"
```

### DigitalOcean Spaces

To back up your database to DigitalOcean Spaces, you'll first need to create a bucket in the DigitalOcean dashboard.
This can be done in the "Spaces Object Storage" tab or by visiting https://cloud.digitalocean.com/spaces.

Once you have created a bucket you will be shown an access key ID and a secret key. Additionally,
while creating the bucket you will have selected a region to host this bucket which has a URL. For example,
if you choose the datacenter region "Amsterdam - Datacenter 3 - AMS3", your region will be `ams3` and
the endpoint will be `ams3.digitalocean.com`.

### Cloudflare R2

To back up your database to Cloudflare R2, you will first need to create a bucket in the Cloudflare dashboard.
You can find this in the “R2” section of the Cloudflare sidebar.

Once you have created a bucket, you will need to create either an Account or User API token with "Object Read & Write" 
permissions. It is also recommended to limit the token’s scope to the bucket. At the end of the process, you will obtain 
an Access Key ID, Secret Access Key, and jurisdiction-specific endpoints for S3 clients in the format
`<accountid>.r2.cloudflarestorage.com`.

```Dotenv
S3_ACCESS_KEY_ID="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
S3_SECRET_KEY="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
S3_ENDPOINT="<accountid>.r2.cloudflarestorage.com"
S3_REGION=""
S3_BUCKET_NAME="haven_backup"
```

### Deprecation warning

The old `aws` and `gcp` backup providers have been deprecated in favor of the new `s3` provider. If you are using the
old providers, please update your `.env` file to use the new `s3` provider. The old providers will be removed in a future
release.

---

[Backup and Restore](backup.md) | [README](../README.md)
