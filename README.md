# Expenses Tracker

Small Go service that receives structured expense data from NanoClaw and stores expenses in Google Sheets.

## Flow

```text
NanoClaw -> Go Finance Service -> Google Sheets API -> Google Sheet row
```

## Configuration

Configure the service with environment variables:

```text
API_KEY=your-secret-key
GOOGLE_SHEET_ID=your-google-sheet-id
GOOGLE_CREDENTIALS_FILE=/absolute/path/to/service-account.json
PORT=8080
TIMEZONE=Asia/Jakarta
SHEET_RANGE=Expenses!A:G
```

Required variables:

```text
API_KEY
GOOGLE_SHEET_ID
GOOGLE_CREDENTIALS_FILE
```

Defaults:

```text
PORT=8080
TIMEZONE=Asia/Jakarta
SHEET_RANGE=Expenses!A:G
```

## Google Sheets Setup

1. Create or use a Google Cloud service account.
2. Download its JSON credentials file.
3. Share the target Google Sheet with the service account email as an editor.
4. Set `GOOGLE_CREDENTIALS_FILE` to the credentials JSON file path.
5. Set `GOOGLE_SHEET_ID` to the spreadsheet ID from the Sheet URL.

## Target Sheet Columns

Use this header row in the target sheet:

```text
Date | Description | Category | Amount | Payment Method | Account/Wallet | ID
```

Keep the header row in place. The service sorts data rows by date and excludes the header from sorting. You can hide the `ID` column in Google Sheets.

## Run Locally

```bash
go run .
```

## Run With Docker

Build the image:

```bash
docker build -t expenses-tracker:latest .
```

Published images are available from GHCR after pushes to `main`:

```bash
docker pull ghcr.io/kevinssheva/expenses-tracker:latest
```

For private packages, authenticate first:

```bash
echo "$CR_PAT" | docker login ghcr.io -u kevinssheva --password-stdin
```

Run it with env vars and a read-only service account mount:

```bash
docker run --rm \
  --env-file .env \
  -e GOOGLE_CREDENTIALS_FILE=/run/secrets/google-service-account.json \
  -v "$(pwd)/service-account.json:/run/secrets/google-service-account.json:ro" \
  -p 8080:8080 \
  expenses-tracker:latest
```

Run it with Docker Compose:

```bash
docker compose up --build -d
```

If your Docker installation uses the legacy Compose command:

```bash
docker-compose up --build -d
```

The Docker image does not include `.env` or `service-account.json`. The Compose file reads `.env` from the host and mounts `service-account.json` as a read-only file inside the container.

Recommended VM layout:

```text
/opt/expenses-tracker/
  docker-compose.yml
  .env
  service-account.json
```

Restrict secret file permissions on the VM:

```bash
chmod 600 .env service-account.json
```

If NanoClaw runs in Docker on the same VM, put both services in the same Compose network and have NanoClaw call `http://expenses-tracker:8080/expenses`. In that setup, remove the `ports` section from `expenses-tracker` unless the service must be reachable from outside Docker.

Health check:

```bash
curl http://localhost:8080/healthz
```

Record an expense:

```bash
curl -X POST http://localhost:8080/expenses \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{
    "date": "2026-05-25",
    "description": "Makan ayam geprek",
    "category": "Food",
    "amount": 35000,
    "payment_method": "QRIS",
    "account_wallet": "GoPay"
  }'
```

Successful response:

```json
{
  "expense": {
    "id": "exp_...",
    "date": "2026-05-25",
    "description": "Makan ayam geprek",
    "category": "Food",
    "amount": 35000,
    "payment_method": "QRIS",
    "account_wallet": "GoPay"
  }
}
```

Get expenses:

```bash
curl http://localhost:8080/expenses \
  -H "X-API-Key: your-secret-key"
```

Filter by ID or date range:

```bash
curl 'http://localhost:8080/expenses?id=exp_...' \
  -H "X-API-Key: your-secret-key"

curl 'http://localhost:8080/expenses?from=2026-05-01&to=2026-05-31' \
  -H "X-API-Key: your-secret-key"
```

Edit an expense:

```bash
curl -X PATCH http://localhost:8080/expenses/exp_... \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{
    "description": "Spotify",
    "category": "Subscription",
    "account_wallet": "BCA"
  }'
```

Delete an expense:

```bash
curl -X DELETE http://localhost:8080/expenses/exp_... \
  -H "X-API-Key: your-secret-key"
```

## Validation

The service validates incoming expenses before writing to Google Sheets:

```text
date is required and must use YYYY-MM-DD
description must not be empty
category must be one of Food, Subscription, Transport, Shopping, Bills, Health, Entertainment, Education, Other
amount must be greater than 0
rows are sorted by date after create and edit
```
