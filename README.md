# Slack Repeated Schedule Sender

A simple Go CLI tool to schedule Slack messages with support for recurring schedules.

## Features

- Schedule one-time messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, emoji, links, etc.)
- Messages sent in Pacific time

## Installation

```bash
# Clone the repo
git clone https://github.com/daggerpov/slack-repeated-schedule-sender.git
cd slack-repeated-schedule-sender

# Build
go build -o slack-scheduler

# Or install globally
go install
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

# Or manually create ~/.slack-scheduler-credentials.json
echo '{"token": "xoxp-your-token-here"}' > ~/.slack-scheduler-credentials.json
chmod 600 ~/.slack-scheduler-credentials.json
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
| `--time` | `-t` | Time to send (HH:MM, 24-hour, Pacific time) |

### Optional Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interval` | `-i` | `none` | Repeat interval: `none`, `daily`, `weekly`, `monthly` |
| `--count` | `-n` | `1` | Number of times to send |
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

## Message Formatting

The message field supports full Slack formatting:

- **Mentions:** `@username`, `@channel`, `@here`
- **Emoji:** `:thumbsup:`, `:rocket:`, `:coffee:`
- **Bold/Italic:** `*bold*`, `_italic_`
- **Links:** `<https://example.com|Click here>`
- **Code:** `` `code` ``, ` ```code block``` `

## Limitations

- Slack only allows scheduling messages up to **120 days** in advance
- Past times are automatically skipped
- The tool schedules messages via Slack's API, so they appear as scheduled messages in Slack

## Credentials File

The tool looks for credentials in:
1. `./.slack-scheduler-credentials.json` (current directory)
2. `~/.slack-scheduler-credentials.json` (home directory)

Format:
```json
{
  "token": "xoxp-your-user-oauth-token"
}
```

⚠️ **Never commit your credentials file!** It's already in `.gitignore`.

## License

MIT