# Google Play Android Review Fetcher

This application fetches reviews from the Google Play Store and sends notifications to a Google Chat webhook.

## Configuration

The application can be configured using environment variables. Here are the available options:

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `PACKAGE_NAME` | Your Android app's package name | `com.example.app` |
| `WEBHOOK_URL` | Google Chat webhook URL for notifications | (default webhook URL) |
| `KEY_PATH` | Path to Google service account credentials | `./service-account.json` |
| `REVIEWS_CSV` | CSV file to store reviews | `reviews.csv` |
| `LOG_DIR` | Directory for log files | `./logs` |
| `TEST_MODE` | When true, webhook messages are only logged and not sent | `true` |

## Setting up Environment Variables

You can set environment variables in several ways:

1. Using a `.env` file (recommended):
   ```bash
   PACKAGE_NAME=com.example.app
   WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/XXXXX/messages?key=YOUR_KEY&token=YOUR_TOKEN
   KEY_PATH=./service-account.json
   REVIEWS_CSV=reviews.csv
   LOG_DIR=./logs
   TEST_MODE=true
   ```

2. Setting them directly in your shell:
   ```bash
   export PACKAGE_NAME=com.example.app
   export WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/XXXXX/messages?key=YOUR_KEY&token=YOUR_TOKEN
   export TEST_MODE=false
   ```

3. Setting them when running the application:
   ```bash
   PACKAGE_NAME=com.example.app WEBHOOK_URL=your_webhook_url TEST_MODE=false ./your_binary
   ```

## Usage

Run the application with optional rating range filter:

```bash
# Run without filters
./your_binary

# Run with rating filter (e.g., only reviews with ratings 1-3)
./your_binary 1-3
```

## Features

- Fetches reviews from Google Play Console
- Filters reviews by rating range (optional)
- Sends new reviews to Google Chat webhook
- Maintains a CSV file of all reviews
- Logs all activities to daily log files

## Prerequisites

- Go 1.21 or later
- Google Play Console API credentials (service account JSON file)
- Google Chat webhook URL

## Installation

1. Clone the repository:
```bash
git clone https://github.com/wilik16/google-play-android-review-fetcher.git
cd google-play-android-review-fetcher
```

2. Install dependencies:
```bash
go mod download
```

3. Place your Google Play Console service account JSON file in the project root directory as `service-account.json`

## Building

To build the binary:

```bash
go build -o review-fetcher
```

The binary can then be run directly:

```bash
# Run without rating filter
./review-fetcher

# Run with rating filter
./review-fetcher 1-3
```

## Output

- Reviews are stored in `reviews.csv`
- Logs are stored in the `logs` directory with daily rotation
- New reviews are sent to the configured Google Chat webhook