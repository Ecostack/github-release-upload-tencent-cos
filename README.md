# Release Asset Downloader

This project downloads release assets from GitHub and uploads them to Tencent Cloud Object Storage (COS). It also notifies a Discord channel when new assets are uploaded.

## Features

- Uses GitHub API to get latest release information for a repository
- Downloads each asset from that release
- Checks if asset already exists in COS bucket before downloading
- Uploads downloaded assets to COS bucket
- Generates public URLs for uploaded assets in COS
- Sends a message to a Discord channel with the asset URL
- Environment variables are used for configuration

## Usage

1. Create a `.env` file and add your environment variables:
```
GITHUB_OWNER={github_owner}
GITHUB_REPO={github_repo}
GITHUB_TOKEN={github_access_token}

COS_BUCKET={cos_bucket}
COS_REGION={cos_region}
COS_SECRET_KEY={cos_secret_key}
COS_SECRET_ID={cos_secret_id}

DISCORD_TOKEN={cos_secret_key}
DISCORD_CHANNEL_ID={discord_channel_id}
```

2. Run the program:
```
go run main.go
```

3. The program will download the latest release assets for the configured GitHub repository and upload them to your COS bucket. It will also send a message in the configured Discord channel with asset URLs.

4. The program will exit once all assets have been processed.

That's the basic usage. Let me know if you want me to expand the README or modify it in any way!