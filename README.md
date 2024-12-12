# GitHub Issue Monitor

WebSocket-based monitoring tool to monitor GitHub issue updates in real time.

## Features

- Real-time monitoring of issues in GitHub repository
- Real-time notification via WebSocket
- Integration with GitHub Webhook
- Support for multiple simultaneous client connections

## Requirements

- wscat (for client testing)

## How to use

1. connecting with a WebSocket client
```bash
wscat -c wss://dda7-20-172-44-136.ngrok-free.app/ws
```

Creating or updating an issue
- Create or update an issue in the repository
- A notification appears in the WebSocket client

## Project Configuration

```
github-issue-monitor/
├── .github/
│ └── workflows/ # GitHub Actions settings
├── cmd/
│ └── server/    # Entry point for server
├── internal/
│ ├── handler/   # WebSocket and Webhook handler
│ └── models/    # Data model
└── README.md
```

## Demo environment

This application is deployed continuously via GitHub Actions and published via ngrok.
The URL is generated at each deployment and can be found in the Actions execution log on GitHub.

## Development Notes

1. local testing
   - Start local server: `go run cmd/server/main.go`.
   - Tunneling with ngrok: `ngrok http 8080`.
   - WebSocket connection test: `wscat -c wss://[ngrok-url]/ws`

2. Update Webhook URL: `wscat -c wss://[ngrok-url]/ws`
   - You need to update your GitHub webhook configuration to the new ngrok URL

## Security precautions

- This implementation is for demonstration purposes only and requires additional security measures for use in a production environment
