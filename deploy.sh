#!/bin/bash

# Install dependencies
echo "Installing dependencies..."
go mod tidy

# Build the binary
echo "Building binary..."
go build -o review-fetcher

# Create deployment directory
echo "Creating deployment directory..."
mkdir -p deploy
cp review-fetcher service-account.json deploy/

# Instructions for deployment
echo "
Deployment files are ready in the 'deploy' directory.
To deploy to your server:

1. Copy files to VPS:
   scp -r deploy/* user@ip.ip.ip.ip:~/google-play-android-review-fetcher/

2. SSH into your server:
   ssh user@ip.ip.ip.ip

3. Set up the binary:
   cd ~/google-play-android-review-fetcher
   chmod +x review-fetcher

4. Set up Cron Job:
   crontab -e
   Add this line to run daily at 10 AM:
   0 10 * * * cd ~/google-play-android-review-fetcher && ./review-fetcher >> ~/google-play-android-review-fetcher/logs/cron.log 2>&1

5. To monitor:
   tail -f ~/google-play-android-review-fetcher/logs/cron.log
   cat ~/google-play-android-review-fetcher/reviews.csv
" 