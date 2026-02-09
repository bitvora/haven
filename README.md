# HAVEN

HAVEN (High Availability Vault for Events on Nostr) is the most sovereign personal relay for the Nostr protocol, for storing and backing up sensitive notes like eCash, private chats and drafts. It is a relay that is not so dumb, with features like web of trust, inbox relay, cloud backups, blastr and the ability to import old notes. It even includes it's own blossom media server!

## Four Relays in One + Blossom Media Server

**Private Relay**: This relay is only accessible by the owner of the relay. It is used for drafts, ecash and other private notes that nobody can read or write to. It is protected by Auth.

**Chat Relay**: This relay is used to contact the owner by DM. Only people in the web of trust can interact with this relay, protected by Auth. It only accepts encrypted DMs and group chat kinds.

**Inbox Relay**: This relay is where the owner of the relay reads from. Send your zaps, reactions and replies to this relay when you're tagging the owner. You can also pull notes from this relay if you want notes where the owner is tagged. This relay automatically pulls notes from other relays. Only notes where the owner is tagged will be accepted to this relay.

**Outbox Relay**: This relay is where the owner's notes all live and are publicly accessible. You can import all your old notes to this relay. All notes sent to this relay are blasted to other relays. Only the owner can send to this relay, but anyone can read.

**Blossom Media Server**: This relay also includes a media server for hosting images and videos. You can upload images and videos to this relay and get a link to share them. Only the relay owner can upload to this relay, but anyone can view the images and videos.

## Not So Dumb Relay Features

**Web of Trust**: Protected from DM and Inbox spam by using a Web of Trust (WoT). See the [Web of Trust Documentation](docs/wot.md) for more details.

**Inbox Relay**: Notes are pulled from other relays and stored in the inbox relay.

**Cloud Backups**: Notes are backed up in the cloud and can be restored if the relay is lost.

**Blastr**: Notes sent to the outbox are also blasted to other relays.

**Import Old Notes**: Import your old notes and notes you're tagged in from other relays.

**JSONL Export/Import**: It is your data, export to JSONL at any time. See the [JSONL Documentation](docs/jsonl.md) for more details.

## Installation

### Option 1: Download Pre-built Binaries (Recommended)

The easiest way to get started with Haven is to download pre-built binaries from our GitHub releases page:

**[Download Haven Releases](https://github.com/bitvora/haven/releases/)**

#### Installation Steps:

1. **Download the appropriate binary** for your system from the releases page
2. **Verify the download (optional)**: See our [Verification Documentation](docs/verify.md) for 
instructions on how to verify the authenticity of the binaries using GPG signatures and checksums.
3. **Create a haven directory** and extract the downloaded file:
   ```bash
   mkdir haven
   # For Linux/macOS:
   tar -xzf haven_[Platform]_[Architecture].tar.gz -C haven
   # For Windows: extract the .zip file to this directory
   ```
### Option 2: Build from Source

If you prefer to build Haven from source or need to customize the build, please see the [Build Documentation](docs/build.md).

## Setup Instructions

Follow these steps to get the Haven Relay running on your local machine (after installing via binary download or building from source):

### 1. Copy `.env.example` to `.env`

You'll need to create an `.env` file based on the example provided.

```bash
cp .env.example .env
```

### 2. Set your environment variables

Open the `.env` file and set the necessary environment variables.

### 3. Create the relays JSON files

Copy the example relays JSON files for your seed and blastr relays:

```bash
cp relays_import.example.json relays_import.json
```

```bash
cp relays_blastr.example.json relays_blastr.json
```

The JSON should contain an array of relay URLs, which default to wss:// if you don't explicitly specify the protocol.

### 4. Run on System Startup

### Linux - Create a Systemd Service
To have the relay run as a service, create a systemd unit file. Make sure to limit the memory usage to less than your system's total memory to prevent the relay from crashing the system.
Replace the values for `ExecStart` and `WorkingDirectory` with the actual paths where you installed Haven and stored the `.env` file.


1. Create the file:

```bash
sudo nano /etc/systemd/system/haven.service
```

2. Add the following contents:

```ini
[Unit]
Description=Haven Relay
After=network.target

[Service]
ExecStart=/home/ubuntu/haven/haven #Edit path to point to where you installed Haven
WorkingDirectory=/home/ubuntu/haven #Edit path to point to where you installed Haven
MemoryMax=1000M  # Example, Limit memory usage to 1000 MB | Edit this to fit your machine
Restart=always

[Install]
WantedBy=multi-user.target
```


3. Reload systemd to recognize the new service:

```bash
sudo systemctl daemon-reload
```

4. Start the service:

```bash
sudo systemctl start haven
```

5. (Optional) Enable the service to start on boot:

```bash
sudo systemctl enable haven
```

### MacOS - Create a login item App
To have the relay run on boot, create a script that will open terminal and run the haven binary, the termainal will remain open and the relay running with it. Be sure the download /haven directory is locacted in the MacOS home folder ~/

1. Create the App: Open Script Editor

2. Add the following contents:

```ini
tell application "Terminal"
  activate
  do script "cd \"$HOME/haven\"; ./haven; exec $SHELL"
end tell
```
3. Save in Applications folder

4. Open System Settings - General - Login Items
   Hit the plus, add run_haven from Applications folder

5. Reboot - On initial restart and terminal auto-open choose “allow”

6. Reboot again to test login item

### 5. Serving over nginx or apache (optional)

To have a domain name (example: relay.domain.com) point to your machine, you will need to setup an nginx.

1. Install nginx on your relay:

```bash
sudo apt-get update 
sudo apt-get install nginx
```

2. Remove default config: `sudo rm -rf /etc/nginx/sites-available/default`

3. Create new default config: `sudo nano /etc/nginx/sites-available/default` 

4. Add new reverse proxy config by adding the following configuration to your nginx configuration file:

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    client_max_body_size 100m;

    location / {
        proxy_pass http://localhost:3355;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

Or apache configuration file:
```apache
<VirtualHost *:80>
        ServerName yourdomain.com

        RewriteEngine On
        RewriteCond %{HTTP:Upgrade} websocket [NC]
        RewriteCond %{HTTP:Connection} upgrade [NC]
        RewriteRule ^/?(.*) "ws://localhost:3355/$1" [P,L]

        # Proxy for HTTP traffic (NIP-11 relay info page)
        ProxyPass / http://localhost:3355/
        ProxyPassReverse / http://localhost:3355/

        # Optional: Add HSTS header for enhanced security
        Header always set Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"

        # Optional: Set appropriate WebSocket headers
        RequestHeader set Upgrade "websocket"
        RequestHeader set Connection "Upgrade"
</VirtualHost>
```

Replace `yourdomain.com` with your actual domain name.

> [!NOTE]
> [`client_max_body_size`](https://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size) is set to 100m
> to allow for larger media files to be uploaded to Blossom. `0` can be used to allow for unlimited file sizes. If you are 
> using Cloudflare proxy, be mindful of [upload limits](https://community.cloudflare.com/t/maximum-upload-size-is-limit/418490/2).

After adding the configuration, restart nginx:

```bash
sudo systemctl restart nginx
```

Apache:
```bash
sudo systemctl restart httpd
```

### Alternative: Serving over Caddy
<details><summary>Click here to view the installation routine for Caddy</summary>
<p>

Preparation: Set the A record (for your domain) to point to the server's IP address.

1. Install caddy:

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

2. Open Caddyfile:

```bash
sudo nano /etc/caddy/Caddyfile
```

3. Add configuration:

```bash
# Configuration for HAVEN Relay
yourdomain.com {
    reverse_proxy localhost:3355 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
        transport http {
            versions 1.1
        }
    }
    request_body {
        max_size 100MB
    }
}
```

4. Reload Caddy:

```bash
sudo systemctl reload caddy
```

5. Check status logs:

```bash
sudo systemctl status caddy
sudo journalctl -u caddy -f --since "2 hour ago"
```

**Note:** Caddy automatically manages certificates and WebSocket connections. Certbot is not required.

</p>
</details>

### 7. Install Certbot (optional)

If you want to serve the relay over HTTPS, you can use Certbot to generate an SSL certificate.

```bash
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx
```

After installing Certbot, run the following command to generate an SSL certificate:

```bash
sudo certbot --nginx
```

Apache:

```bash
sudo certbot --apache
```

Follow the instructions to generate the certificate.

Note: Command will fail if the Domain you added to nginx is not yet pointing at your machine's IP address. 
This is done by adding an A record subdomain pointing to your IP address through your DNS recrods Manager.

### 8. Run The Import (optional)

If you want to import your old notes and notes you're tagged in from other relays, run the following command:

```bash
sudo systemctl stop haven
./haven --import
sudo systemctl start haven
```

### 9. Access the relay

Once everything is set up, the relay will be running on `localhost:3355` with the following endpoints:

- `localhost:3355` (outbox and Blossom server)
- `localhost:3355/private`
- `localhost:3355/chat`
- `localhost:3355/inbox`

## Database

Haven currently supports [BadgerDB](https://github.com/dgraph-io/badger) and [LMDB](https://www.symas.com/mdb) as embedded
databases, meaning no external database is required.

By default, Haven uses BadgerDB. To switch to LMDB, set the `DB_ENGINE` environment variable to `lmdb` in the `.env` file.

LMDB can be faster than BadgerDB but performs best with NVMe drives and may require fine-tuning based on factors such as
database size, operating system, file system, and hardware.

### LMDB Map Size

There is no one-size-fits-all value for LMDB’s map size. Windows and macOS users, in particular, may need
to adjust the `LMDB_MAPSIZE` environment variable to a value lower than the available free disk space if the default
value of 273 GB is too high. Otherwise, Haven will fail to bootstrap. Users with large databases may also need to
increase the `LMDB_MAPSIZE` value above the default. On most systems, the default value should work fine.

Despite the large default value, on most modern systems LMDB will only use the disk space it needs. The map size simply
defines an upper limit for the database size. For more information about LMDB’s map size, refer to the
[LMDB documentation](http://www.lmdb.tech/doc/group__mdb.html#gaa2506ec8dab3d969b0e609cd82e619e5).

### Migrating from databases created in older versions of Haven

Haven uses [Khatru's event store](https://github.com/fiatjaf/eventstore) to store notes. The way events are stored evolves 
over time, and occasionally this introduces breaking changes.

As a precaution, before upgrading to a newer version of Haven, you should back up the `db` folder.

Haven versions 1.0.3 and earlier did not replace outdated notes. While this does not affect the relay's core
functionality, it can result in a bloated database, reduced performance, and bugs in some clients. For this reason, it
is recommended to delete old databases and start fresh.

BadgerDB users upgrading from Haven version 1.0.5 or earlier may encounter a critical error when starting the relay:

```
error running migrations: failed to delete index key xxxx: Txn is too big to fit into one request
```

As a workaround, you can delete the `db` folder and start fresh, optionally [re-importing](#8-run-the-import-optional)
previous notes.

## Blossom Media Server

The outbox relay also functions as a media server for hosting images and videos. You can upload media files to the relay and obtain a shareable link.  
Only the relay owner has upload permissions to the media server, but anyone can view the hosted images and videos.

Media files are stored in the file system based on the `BLOSSOM_PATH` environment variable set in the `.env` file. The default path is `./blossom`.

## Cloud Backups

The relay automatically backs up your database to a cloud provider of your choice.

### S3-Compatible Object Storage

To back up your database to S3 compatible storage such as [AWS S3](https://aws.amazon.com/s3/), 
[GCP Cloud Storage] or 
[DigitalOcean Spaces](https://www.digitalocean.com/products/spaces).

First need to create the bucket on your provider. After creating the Bucket you will be provided with:

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

#### AWS S3

For AWS S3, set the appropriate endpoint for your region/availability zone:

```Dotenv
S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
S3_ENDPOINT="s3.us-east-1.amazonaws.com""
S3_REGION="us-east-1"
S3_BUCKET_NAME="haven_backup"
```

#### GCP Cloud Storage

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

#### DigitalOcean Spaces

To back up your database to DigitalOcean Spaces, you'll first need to create a bucket in the DigitalOcean dashboard.
This can be done in the "Spaces Object Storage" tab or by visiting https://cloud.digitalocean.com/spaces.

Once you have created a bucket you will be shown an access key ID and a secret key. Additionally,
while creating the bucket you will have selected a region to host this bucket which has a URL. For example,
if you choose the datacenter region "Amsterdam - Datacenter 3 - AMS3", your region will be `ams3` and
the endpoint will be `ams3.digitalocean.com`.

### Deprecation warning

The old `aws` and `gcp` backup providers have been deprecated in favor of the new `s3` provider. If you are using the
old providers, please update your `.env` file to use the new `s3` provider. The old providers will be removed in a future
release.

## License

This project is licensed under the [MIT](./LICENSE) License.
