# Slack CLI

Slack CLI is a command-line tool to interact with Slack, allowing users to send messages, manage reactions, upload/download files, and more directly from the terminal.

![image](./image.png)

## Features

- Send messages to Slack channels
- Fetch and display messages from Slack channels
- Upload and download files
- Manage reactions (add/remove)
- Update and delete messages
- Choose channels

## Run Binary
For direct execution, please refer to the [Releases](https://github.com/junnushon/slack-cli/releases/tag/v0.1)

## Installation

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.16 or higher)

### Clone the Repository

```sh
git clone https://github.com/junnushon/slack-cli.git
cd slack-cli
```

## Build the Project
### To build the project for multiple platforms:

```sh

make all
```

### To build for a specific platform:

```sh
make build-linux
# or
make build-windows
# or
make build-darwin
```
## Configuration
Before running the CLI, you need to configure it:

enter your Slack credentials and channel information in the slack.config.json file.  
slack.emoji.json is used to display emoji in messages. 

Both files must always be in the same folder as the binary file.

```json

{
    "slack_bot_token": "your_slack_bot_token",
    "slack_user_token": "your_slack_user_token",
    "channel_id": "your_channel_id",
    "user_cache": {
        "U075JAXRYV7": "ServerBot"
    },
    "default_show_limit": 20,
    "default_emoji": "white-check-mark"
}
```
slack_user_token : Required  
slack_bot_token : Optional (If not provided, user_token will be used.)

Required Slack API OAuth Scope (User) :  
- channels:history  
- channels:read  
- chat:write  
- files:read  
- files:write  
- groups:history  
- groups:read  
- im:history  
- im:read  
- links:write  
- mpim:history  
- mpim:read  


## Usage
### Show Commands
```sh

./slack show
./slack show 100
./slack show --date 2023-12-31
./slack show --date 2023-12-29:2023-12-31
./slack show --search keyword
./slack show 500 --search keyword
./slack show --filter keyword
./slack show 500 --filter keyword
./slack show --files
```
### Send Message
```sh

./slack send "Hello, Slack!"
./slack send "Hello, Slack!" --ts 1234567890.123456 (reply)
```
### Edit Message
```sh

./slack edit 1234567890.123456 "Updated message"
```
### Delete Message
```sh

./slack delete 1234567890.123456
```
### Upload File
```sh

./slack upload "path/to/your/file.txt"
```
### Download File
```sh

./slack download "https://file.url"
```
### Manage Reactions
```sh

./slack emoji
./slack emoji 1234567890.123456
./slack emoji 1234567890.123456 white-check-mark
./slack emoji 1234567890.123456 --add thumbsup
./slack emoji 1234567890.123456 --del white-check-mark
```
### Choose Channels
```sh

./slack channels
./slack channels --current
./slack channels --current channel_name
```
### Show Examples
```sh

./slack examples
```
## License
This project is licensed under the MIT License - see the LICENSE file for details.

