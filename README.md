# Slack Recurring Messages Scheduler

A simple Go CLI tool to schedule Slack messages with support for recurring schedules.

## Features

- Schedule one-time messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, emoji, links, etc.)
- Uses your system's local timezone

## Installation

```bash
# Clone the repo
git clone https://github.com/daggerpov/slack-recurring-messages-scheduler.git
cd slack-recurring-messages-scheduler

# Build
make build

# Or install globally
make install
```

## Project Structure

This project follows the [Go standard project layout](https://github.com/golang-standards/project-layout):

```
.
├── cmd/
│   └── slack-scheduler/    # Main application entry point
│       ├── main.go
│       └── main_test.go
├── internal/               # Private application code
│   ├── config/             # Configuration & credentials handling
│   ├── scheduler/          # Scheduling logic
│   ├── slack/              # Slack API client wrapper
│   └── types/              # Shared type definitions
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Setup

### 1. Create a Slack App

1. Go to [Slack API Apps](https://api.slack.com/apps)
2. Click "Create New App" → "From scratch"
3. Name it (e.g., "Message Scheduler") and select your workspace

### 2. Configure Permissions

1. Go to "OAuth & Permissions" in the sidebar
2. Under "User Token Scopes", add:
   - `chat:write` - Send messages as yourself
   - `channels:read` - Read channel info (to resolve names)
   - `groups:read` - Read private channel info

3. Click "Install to Workspace" and authorize

### 3. Get Your Token

1. After installing, copy the "User OAuth Token" (starts with `xoxp-`)
2. Create a credentials file:

```bash
# Create template
./slack-scheduler init

# Or manually create .slack-scheduler-credentials.json
echo '{"token": "xoxp-your-token-here"}' > .slack-scheduler-credentials.json
chmod 600 .slack-scheduler-credentials.json
```

## Usage

```bash
slack-scheduler [flags]
```

### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--message` | `-m` | Message to send (supports @mentions, emoji, formatting) |
| `--channel` | `-c` | Channel name or ID |
| `--date` | `-d` | Start date (YYYY-MM-DD) |
| `--time` | `-t` | Time to send (HH:MM, 24-hour, local time) |

### Optional Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interval` | `-i` | `none` | Repeat interval: `none`, `daily`, `weekly`, `monthly` |
| `--count` | `-n` | `1` | Number of times to send |
| `--end-date` | `-e` | | End date (YYYY-MM-DD). Recurrence stops on or before this date |
| `--days` | | | Days of week (comma-separated: `mon,tue,wed,thu,fri,sat,sun`) |

### Examples

**One-time message:**
```bash
slack-scheduler -m "Hello team!" -c general -d 2025-01-17 -t 14:00
```

**Every Friday at 2pm for 4 weeks:**
```bash
slack-scheduler \
  -m "Weekly reminder: Please submit your timesheets!" \
  -c general \
  -d 2025-01-17 \
  -t 14:00 \
  -i weekly \
  -n 4
```

**Monday and Friday at 9am for 8 occurrences:**
```bash
slack-scheduler \
  -m "Standup time! :coffee:" \
  -c engineering \
  -d 2025-01-13 \
  -t 09:00 \
  -i weekly \
  -n 8 \
  --days mon,fri
```

**Daily reminder for 5 days:**
```bash
slack-scheduler \
  -m "@channel Don't forget to check your PRs" \
  -c dev-team \
  -d 2025-01-13 \
  -t 10:00 \
  -i daily \
  -n 5
```

**Monthly report reminder:**
```bash
slack-scheduler \
  -m "Monthly metrics report due this week" \
  -c analytics \
  -d 2025-01-01 \
  -t 09:00 \
  -i monthly \
  -n 12
```


**Sundays until April 10th (stops at last Sunday on or before end date):**
```bash
slack-scheduler \
  -m "Weekly Sunday update" \
  -c team-updates \
  -d 2025-01-05 \
  -t 10:00 \
  -i weekly \
  --days sun \
  -e 2025-04-10
```

**Daily messages until a specific date:**
```bash
slack-scheduler \
  -m "Daily standup reminder" \
  -c engineering \
  -d 2025-01-13 \
  -t 09:00 \
  -i daily \
  -e 2025-01-31
```


## Message Formatting

The message field supports full Slack formatting:

- **Mentions:** `@username`, `@channel`, `@here`
- **Emoji:** `:thumbsup:`, `:rocket:`, `:coffee:`
- **Bold/Italic:** `*bold*`, `_italic_`
- **Links:** `<https://example.com|Click here>`
- **Code:** `` `code` ``, ` ```code block``` `

## Managing Scheduled Messages

### List Scheduled Messages

View all messages you've scheduled via the API:

```bash
# List all scheduled messages
slack-scheduler list

# List scheduled messages for a specific channel
slack-scheduler list -c general
```

### Delete Scheduled Messages

Cancel scheduled messages:

```bash
# Delete a specific scheduled message by ID
slack-scheduler delete -c general --id Q0A7Z0QMWAF

# Delete ALL scheduled messages in a channel
slack-scheduler delete -c general --all
```

## Important: Slack UI Limitation ⚠️ **Messages scheduled via the Slack API do NOT appear in Slack's "Scheduled Messages" UI.**

This is a Slack platform limitation, not a bug. Here's what this means:

| Scheduled via | Visible in Slack UI | Actually sends |
|--------------|---------------------|----------------|
| Slack app (typing /schedule or clicking schedule) | ✅ Yes | ✅ Yes |
| This CLI tool (API) | ❌ No | ✅ Yes |

**Your messages ARE scheduled and WILL be sent** — you just can't see them in Slack's desktop/mobile app.

To view and manage API-scheduled messages, use:
- `slack-scheduler list` — see all scheduled messages
- `slack-scheduler delete` — cancel scheduled messages

## Limitations

- Slack only allows scheduling messages up to **120 days** in advance
- Past times are automatically skipped
- API-scheduled messages don't appear in Slack's UI (see above), but they will still be sent on schedule

## Credentials File

The credentials file should be in the project directory:
- `./.slack-scheduler-credentials.json`

Format:
```json
{
  "token": "xoxp-your-user-oauth-token"
}
```

**Never commit your credentials file!** It's already in `.gitignore`.

## License

MIT
