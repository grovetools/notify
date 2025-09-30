## Configuration

Configuration for `notify` is handled through command-line flags passed to its subcommands. The tool does not currently read configuration from environment variables or dedicated configuration files.

### Notification Backends

The tool supports two notification backends, selected by the subcommand used:

*   **`system`**: Utilizes the host operating system's native notification mechanism. It executes `osascript` on macOS and `notify-send` on Linux.
*   **`ntfy`**: Sends an HTTP POST request to a specified [ntfy.sh](https://ntfy.sh) server instance.

### System Notification Configuration

The `notify system` command accepts the following flags:

*   `--title <string>`: Sets the notification title. Defaults to "Grove Notification".
*   `--level <string>`: Sets the notification level, which is prepended to the title. Accepted values are `info`, `warning`, `error`. Defaults to `info`.

### Ntfy Notification Configuration

The `notify ntfy` command sends metadata to the ntfy server via HTTP headers. It accepts the following flags:

*   `--url <string>`: The base URL of the ntfy server. Defaults to `https://ntfy.sh`.
*   `--topic <string>`: The ntfy topic to publish the message to. This flag is required.
*   `--title <string>`: Sets the notification title via the `Title` HTTP header. Defaults to "Grove Notification".
*   `--priority <string>`: Sets the message priority via the `Priority` HTTP header. Accepted values are `min`, `low`, `default`, `high`, `urgent`. Defaults to `default`.
*   `--tags <string>`: A comma-separated list of tags sent via the `Tags` HTTP header (e.g., `prod,deploy`).

### Examples

#### System Notification

To send a local system notification about a failed build:
```bash
notify system --title "Build Failed" --level error "Compilation failed in package main."
```

#### Remote Ntfy Notification

To send a notification to an ntfy topic for a successful deployment event:
```bash
notify ntfy \
  --topic "ci-cd-events" \
  --title "Deployment Succeeded" \
  --priority high \
  --tags "prod,deploy" \
  "Version v1.4.1 deployed to production cluster."
```

#### Self-Hosted Ntfy Instance

To send a notification to a self-hosted ntfy server:
```bash
notify ntfy \
  --url "http://ntfy.internal.corp" \
  --topic "dev-alerts" \
  "Database migration complete on staging."
```