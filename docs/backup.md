# Backup and Restore

## Manual Backup and Restore

If you want to backup all relay data to a JSONL zip file, run the following command:

```bash
./haven backup
```

This will create a `haven_backup.zip` file in your current directory. You can specify a different filename:

```bash
./haven backup mybackup.zip
```

To backup a specific relay to a JSONL file:

```bash
./haven backup --relay outbox outbox.jsonl
```

To restore data from a `haven_backup.zip` file, run:

```bash
./haven restore haven_backup.zip
```

To restore a specific relay from a JSONL file:

```bash
./haven restore --relay outbox outbox.jsonl
```

## Periodic Cloud Backups

Haven can periodically back up your data to a cloud provider of your choice.

To back up your database to S3 compatible storage such as [AWS S3](https://aws.amazon.com/s3/), 
[GCP Cloud Storage](https://cloud.google.com/storage), 
[DigitalOcean Spaces](https://www.digitalocean.com/products/spaces) or
[Cloudflare R2](https://www.cloudflare.com/developer-platform/r2/).

First you need to create the bucket on your provider. After creating the Bucket you will be provided with:

- Access Key ID
- Secret Key
- URL Endpoint
- Region
- Bucket Name

Once you have this data, update your `.env` file with the appropriate information:

```Dotenv
S3_ACCESS_KEY_ID="your_access_key_id"
S3_SECRET_KEY="your_secret_key"
S3_ENDPOINT="your_endpoint"
S3_REGION="your_region"
S3_BUCKET_NAME="your_bucket"
```

Replace `your_access_key_id`, `your_secret_access_key`, `your_region`, and `your_bucket` with your actual credentials.

You may also want to set the `BACKUP_INTERVAL_HOURS` environment variable to specify how often the relay should back up 
the database.

```Dotenv
BACKUP_INTERVAL_HOURS=24
```

Finally, you need to specifiy `s3` as the backup provider:

```Dotenv
BACKUP_PROVIDER="s3" # s3, none (or leave blank to disable)
```

See [Cloud Storage Provider Specific Instructions](cloud-storage.md) for more details.

---

[README](../README.md)
