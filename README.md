# Confluence Page Backup Tool

This tool provides the functionality to fetch Confluence pages through the Confluence API, convert the page contents from HTML to Markdown, and save the data locally. Additionally, images found in the page contents are downloaded and saved locally as well.

## Prerequisites

- Go (Golang) installed.
- Confluence API URL, email, and API token for authentication.

## Usage

### Command-Line Arguments

- `--backup-dir`: Directory to save backups (default: `./backup`).
- `--api-url`: Base URL of the Confluence API (required).
- `--email`: Email address for API authentication (required).
- `--api-token`: API token for authentication (required).
- `--page-ids`: List of Confluence page IDs to fetch (required).


## Installation

1. Clone the repository:

```sh
git clone https://github.com/yourusername/confluence-page-backup.git
cd confluence-page-backup
```

2. Install dependencies and build the project:

```sh
go mod tidy
```

### Usage Example

Here's an example of how to run the tool:

```sh
go run main.go \
  --backup-dir ./backup \
  --api-url "https://your-confluence-instance.atlassian.net/wiki/rest/api" \
  --email "your-email@example.com" \
  --api-token "your-api-token" \
  --page-ids "12345,67890"
```
