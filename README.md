# Slack Recurring Messages Scheduler

A Go CLI tool to schedule Slack messages with support for recurring schedules.

## Demo

<h1 align="center">
  Full Video on YouTube: <a href="https://youtu.be/ZcLI_l6oBdw">https://youtu.be/ZcLI_l6oBdw
</a>
</h1>

<p align="center">
  <a href="https://youtu.be/ZcLI_l6oBdw">
    <img src="https://github.com/user-attachments/assets/1efa68db-4cb1-4b65-893a-64c572edf062" alt="Slack Repeated Schedule Sender Demo" width="1000" height="auto">
  </a>
</p>

<p align="center">
   Sorry for the lack of color in this clip. I used <a href="https://en.wikipedia.org/wiki/FFmpeg">ffmpeg</a> to compress this clip from the <b>full video above</b> into a gif and had to choose between fps, resolution, and color 
   <br>-> I chose resolution for legibility.
</p>


`./slack-scheduler help` output:

```

A CLI tool to schedule Slack messages with support for:
- One-time scheduled messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, #channel links, emoji, etc.)

Messages are scheduled using your system's local timezone.

IMPORTANT: @channel, @here, and @everyone mentions are automatically converted
to the proper Slack API format to ensure notifications are sent.

Usage:
  ./slack-scheduler [flags]
  ./slack-scheduler [command]

Examples:
  # Send a one-time message
  ./slack-scheduler -m "Hello team!" -c general -d 2025-01-17 -t 14:00

  # Send weekly on Fridays until end date (start date defaults to today)
  ./slack-scheduler -m "Weekly reminder!" -c general -t 14:00 -i weekly --days fri -e 2025-04-01

  # Send on Monday and Friday at 9am for 8 occurrences
  ./slack-scheduler -m "Meeting happening now!" -c engineering -d 2025-01-13 -t 09:00 -i weekly -n 8 --days mon,fri

  # @channel, @here, @everyone, @mentions work
  # So do #channel mentions:
  ./slack-scheduler -m "@channel Hey, please check #meetings." -c general -d 2025-01-17 -t 09:00

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  delete      Delete scheduled messages
  help        Help about any command
  init        Create a credentials template file
  list        List all scheduled messages
  modify      Modify scheduled messages

Flags:
  -c, --channel string    Channel name or ID to send to
  -n, --count int         Max number of times to send (0 = unlimited, use --end-date to limit)
  -d, --date string       Start date (YYYY-MM-DD)
      --days string       Days of week for weekly schedule (comma-separated: mon,tue,wed,thu,fri,sat,sun)
  -e, --end-date string   End date (YYYY-MM-DD). Recurrence stops on or before this date
  -h, --help              help for ./slack-scheduler
  -i, --interval string   Repeat interval: none, daily, weekly, monthly (default "none")
  -m, --message string    Message to send (supports @mentions, #channel links, emoji, Slack formatting)
  -t, --time string       Time to send (HH:MM, 24-hour format, local time)

Use "./slack-scheduler [command] --help" for more information about a command.
```

## Features

- Schedule one-time messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, #channel links, emoji, etc.)
- **@channel, @here, @everyone mentions that actually send notifications** (automatically converted to Slack API format)
- **#channel links that are clickable** (automatically converted to Slack API format)
- Message groups for easy batch management
- Modify scheduled messages (change channel, message, time, etc.)
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
./slack-scheduler [flags]
```

### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--message` | `-m` | Message to send (supports @mentions, #channel links, emoji, formatting) |
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
| `--group` | `-g` | | Name for this message group (for easier management) |

### Examples

**One-time message:**
```bash
./slack-scheduler -m "Hello team!" -c general -d 2025-01-17 -t 14:00
```

**Every Friday at 2pm for 4 weeks:**
```bash
./slack-scheduler \
  -m "Weekly reminder: Please submit your timesheets!" \
  -c general \
  -d 2025-01-17 \
  -t 14:00 \
  -i weekly \
  -n 4
```

**Monday and Friday at 9am for 8 occurrences:**
```bash
./slack-scheduler \
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
./slack-scheduler \
  -m "@channel Don't forget to check your PRs" \
  -c dev-team \
  -d 2025-01-13 \
  -t 10:00 \
  -i daily \
  -n 5
```

**Monthly report reminder:**
```bash
./slack-scheduler \
  -m "Monthly metrics report due this week" \
  -c analytics \
  -d 2025-01-01 \
  -t 09:00 \
  -i monthly \
  -n 12
```


**Sundays until April 10th (stops at last Sunday on or before end date):**
```bash
./slack-scheduler \
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
./slack-scheduler \
  -m "Daily standup reminder" \
  -c engineering \
  -d 2025-01-13 \
  -t 09:00 \
  -i daily \
  -e 2025-01-31
```


## Message Formatting

The message field supports full Slack formatting:

- **Mentions:** `@username`, `@channel`, `@here`, `@everyone`
- **Channel links:** `#channel-name` (automatically converted to clickable links)
- **Emoji:** `:thumbsup:`, `:rocket:`, `:coffee:`
- **Bold/Italic:** `*bold*`, `_italic_`
- **Links:** `<https://example.com|Click here>`
- **Code:** `` `code` ``, ` ```code block``` `

### @channel, @here, @everyone Notifications

Unlike raw Slack API calls, this tool **automatically converts** broadcast mentions to the proper Slack API format:

- `@channel` → `<!channel>` (notifies everyone in the channel)
- `@here` → `<!here>` (notifies active members)
- `@everyone` → `<!everyone>` (notifies everyone in the workspace)

This means you can write natural messages like:
```bash
./slack-scheduler -m "@channel Don't forget standup!" -c general -d 2025-01-17 -t 09:00
```

And recipients will actually receive notifications, just like when you type `@channel` in Slack directly.

**Note:** For @channel/@here/@everyone to work, you must use a **User OAuth Token** (`xoxp-...`), not a Bot Token (`xoxb-...`).

### #channel Links

Channel references are also automatically converted to clickable Slack links:

- `#general` → `<#C1234567|general>` (clickable channel link)
- `#dev-team` → `<#C7654321|dev-team>` (clickable channel link)

Example:
```bash
./slack-scheduler -m "Please post updates in #engineering" -c general -d 2025-01-17 -t 09:00
```

The `#engineering` reference will become a clickable link in the message. If the channel doesn't exist, the original text is preserved.

## Managing Scheduled Messages

### List Scheduled Messages

View all messages you've scheduled via the API:

```bash
# List all scheduled messages
./slack-scheduler list

# List scheduled messages for a specific channel
./slack-scheduler list -c general
```

### Delete Scheduled Messages

Cancel scheduled messages:

```bash
# Delete a specific scheduled message by ID
./slack-scheduler delete -c general --id Q0A7Z0QMWAF

# Delete all messages in a group
./slack-scheduler delete --group "standup-reminders"

# Delete ALL scheduled messages in a channel
./slack-scheduler delete -c general --all
```

### Modify Scheduled Messages

Change attributes of existing scheduled messages. Since the Slack API doesn't support updating scheduled messages directly, this command deletes and recreates them with the new parameters.

```bash
# Change the channel for all messages in a group
./slack-scheduler modify --group "standup" --channel new-channel

# Change the message text for a group
./slack-scheduler modify --group "standup" --message "New message text @channel"

# Change the time for all messages in a group
./slack-scheduler modify --group "standup" --time 10:00

# Modify a single scheduled message
./slack-scheduler modify --id Q0A7Z0QMWAF --channel general --message "Updated text"
```

### Message Groups

When you schedule recurring messages, they're automatically saved as a "group" for easier management. You can also specify a custom group name:

```bash
# Create a group with a custom name
./slack-scheduler -m "Standup time!" -c engineering -d 2025-01-13 -t 09:00 -i weekly -n 8 -g "standup-reminders"

# List all groups
./slack-scheduler list --groups

# Modify all messages in a group at once
./slack-scheduler modify --group "standup-reminders" --channel new-engineering

# Delete all messages in a group
./slack-scheduler delete --group "standup-reminders"
```

Groups are stored locally in `.slack-scheduler-groups.json` and track which scheduled message IDs belong together.

## Important: Slack UI Limitation ⚠️ **Messages scheduled via the Slack API do NOT appear in Slack's "Scheduled Messages" UI.**

This is a Slack platform limitation, not a bug. Here's what this means:

| Scheduled via | Visible in Slack UI | Actually sends |
|--------------|---------------------|----------------|
| Slack app (typing /schedule or clicking schedule) | ✅ Yes | ✅ Yes |
| This CLI tool (API) | ❌ No | ✅ Yes |

**Your messages ARE scheduled and WILL be sent** — you just can't see them in Slack's desktop/mobile app.

To view and manage API-scheduled messages, use:
- `./slack-scheduler list` — see all scheduled messages
- `./slack-scheduler delete` — cancel scheduled messages

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