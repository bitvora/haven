# HAVEN

HAVEN (High Availability Vault for Events on Nostr) is the most sovereign personal relay for the Nostr protocol, for storing and backing up sensitive notes like eCash, private chats and drafts. It is a relay that is not so dumb, with features like web of trust, inbox relay, cloud backups, blastr and the ability to import old notes. It even includes it's own blossom media server!

## Four Relays in One + Blossom Media Server

**Private Relay**: This relay is only accessible by the owner of the relay. It is used for drafts, ecash and other private notes that nobody can read or write to. It is protected by Auth.

**Chat Relay**: This relay is used to contact the owner by DM. Only people in the web of trust can interact with this relay, protected by Auth. It only accepts encrypted DMs and group chat kinds.

**Inbox Relay**: This relay is where the owner of the relay reads from. Send your zaps, reactions and replies to this relay when you're tagging the owner. You can also pull notes from this relay if you want notes where the owner is tagged. This relay automatically pulls notes from other relays. Only notes where the owner is tagged will be accepted to this relay.

**Outbox Relay**: This relay is where the owner's notes all live and are publicly accessible. You can import all your old notes to this relay. All notes sent to this relay are blasted to other relays. Only the owner can send to this relay, but anyone can read.

**Blossom Media Server**: This relay also includes a media server for hosting images and videos. You can upload images and videos to this relay and get a link to share them. Only the relay owner can upload to this relay, but anyone can view the images and videos.

## Not So Dumb Relay Features

**Web of Trust**: Protected from DM and Inbox spam by using a web of trust.

**Inbox Relay**: Notes are pulled from other relays and stored in the inbox relay.

**Cloud Backups**: Notes are backed up in the cloud and can be restored if the relay is lost.

**Blastr**: Notes sent to the outbox are also blasted to other relays.

**Import Old Notes**: Import your old notes and notes you're tagged in from other relays.

## Prerequisites

- **Go**: Ensure you have Go installed on your system. You can download it from [here](https://golang.org/dl/).
- **Build Essentials**: If you're using Linux, you may need to install build essentials. You can do this by running `sudo apt install build-essential`.

## Setup Instructions

Follow these steps to get the Haven Relay running on your local machine:

### 1. Clone the repository

```bash
git clone https://github.com/bitvora/haven.git
cd haven
```

### 2. Copy `.env.example` to `.env`

You'll need to create an `.env` file based on the example provided in the repository.

```bash
cp .env.example .env
```

### 3. Set your environment variables

Open the `.env` file and set the necessary environment variables.

### 4. Create the relays JSON files

Copy the example relays JSON files for your seed and blastr relays:

```bash
cp relays_import.example.json relays_import.json
```

```bash
cp relays_blastr.example.json relays_blastr.json
```

The JSON should contain an array of relay URLs, which default to wss:// if you don't explicitly specify the protocol.

### 4. Build the project

Run the following command to build the relay:

```bash
go build
```

### 5. Create a Systemd Service

To have the relay run as a service, create a systemd unit file. Make sure to limit the memory usage to less than your system's total memory to prevent the relay from crashing the system.

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
ExecStart=/home/ubuntu/haven/haven
WorkingDirectory=/home/ubuntu/haven
Restart=always

[Install]
WantedBy=multi-user.target
```

Replace the values for `ExecStart` and `WorkingDirectory` with the actual paths where you cloned the repository and stored the `.env` file.

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

### 6. Serving over nginx (optional)

You can serve the relay over nginx by adding the following configuration to your nginx configuration file:

```nginx
server {
    listen 80;
    server_name yourdomain.com;

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

Replace `yourdomain.com` with your actual domain name.

After adding the configuration, restart nginx:

```bash
sudo systemctl restart nginx
```

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

Follow the instructions to generate the certificate.

### 8. Run The Import (optional)

If you want to import your old notes and notes you're tagged in from other relays, run the following command:

```bash
./haven --import
```

### 9. Access the relay

Once everything is set up, the relay will be running on `localhost:3355` with the following endpoints:

- `localhost:3355` (outbox and Blossom server)
- `localhost:3355/private`
- `localhost:3355/chat`
- `localhost:3355/inbox`

## Start the Project with Docker Compose

To start the project using Docker Compose, follow these steps:

1. Ensure Docker and Docker Compose are installed on your system.
2. Navigate to the project directory.
3. Ensure the `.env` file is present in the project directory and has the necessary environment variables set. It's recommended to modify the `EMAIL` to a real email address.
4. You'll also need to expose ports 80 and 443 to the internet and set up your DNS A and AAAA (if you are using IPv6)
   records to point to your server's IP address.
6. You can also change the paths of the `db` folder in the `compose.yml` file.

   ```yaml
   volumes:
     - "./db:/app/db" # only change the left side before the colon
   ```

6. Run the following command:

   ```sh
   # in foreground
   docker compose up --build
   # in background
   docker compose up --build -d
   ```

7. For updating the relay, run the following command:

   ```sh
   
   git pull
   docker compose down
   docker compose build --no-cache
   # in foreground
   docker compose up
   # in background
   docker compose up -d
   ```
## Blossom Media Server

The outbox relay also functions as a media server for hosting images and videos. You can upload media files to the relay and obtain a shareable link.  
Only the relay owner has upload permissions to the media server, but anyone can view the hosted images and videos.

Media files are stored in the file system based on the `BLOSSOM_PATH` environment variable set in the `.env` file. The default path is `./blossom`.

## Cloud Backups

The relay automatically backs up your database to a cloud provider of your choice.

### AWS

To back up your database to AWS, you'll need to first install and configure the awscli. You can do this by running the following commands:

```bash
sudo python3 -m pip install awscli
aws configure
```

After configuring the awscli, you can set the following environment variables in your `.env` file:

```bash
AWS_ACCESS_KEY_ID=your_access_key_id
AWS_SECRET_ACCESS_KEY=your_secret_access_key
AWS_REGION=your_region
AWS_BUCKET=your_bucket
```

Replace `your_access_key_id`, `your_secret_access_key`, `your_region`, and `your_bucket` with your actual AWS credentials.

### GCP

To back up your database to GCP, you'll need set up Application Default Credentials (ADC). There are many ways to do so and it varies on the environment you're running the relay on. Check out the [official documentation](https://cloud.google.com/docs/authentication/provide-credentials-adc) for more information.

After authenticating to GCP, set the environment variable below in your `.env` file:

```bash
GCP_BUCKET_NAME="backups"
```

Replace the name of the bucket accordingly.

## License

This project is licensed under the MIT License.
