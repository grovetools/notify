<!-- DOCGEN:OVERVIEW:START -->

`notify` is a command-line tool and Go library for dispatching local system alerts and remote push notifications. It abstracts operating system specifics and remote API details, providing a unified interface for signaling events within the Grove ecosystem.

## Core Mechanisms

**System Abstraction**: The tool detects the host operating system to execute the appropriate native command for desktop notifications:
*   **macOS**: Executes `osascript` to trigger AppleScript notifications.
*   **Linux**: Executes `notify-send` via `libnotify`.

**Remote Dispatch**: Supports `ntfy` for sending push notifications to mobile devices or remote desktops. It constructs HTTP POST requests with support for priorities, tags, and titles, targeting either the public `ntfy.sh` service or self-hosted instances.

**Configuration Integration**: Uses `grove core` to read configuration from `grove.yml`. Settings are defined under the `notifications` extension key, allowing project-specific or user-specific overrides for topics and filtering levels.

## Features

*   **`notify system`**: triggers a local desktop notification. Useful for signaling the completion of long-running shell scripts.
*   **`notify ntfy`**: sends a payload to an ntfy topic. Supports flags for `--priority`, `--tags`, and `--title`.

## Integration with Grove Hooks

`notify` serves as the alerting layer for `grove hooks`.

**Lifecycle Alerts**: When an interactive agent (managed by `flow` or `hooks`) changes state, `hooks` utilizes the `notify` package to alert the user.
*   **Job Ready**: When a background job (e.g., `headless_agent`) completes or pauses for user input (`pending_user`), a notification is triggered.
*   **Agent Idle**: When an interactive agent finishes a turn and awaits instructions, `notify` signals that the terminal requires attention.

**Context Propagation**: Notifications sent via `hooks` include metadata derived from the session context, such as the Plan Name, Job Title, and Repository/Worktree, allowing users to distinguish between multiple concurrent agent sessions.

## Configuration

Configuration is managed via `grove.yml` under the `notifications` extension.

```yaml
extensions:
  notifications:
    ntfy:
      enabled: true
      topic: "my-private-topic"
      url: "https://ntfy.sh" # Optional, defaults to official server
    system:
      levels: ["error", "warning"] # Only show system alerts for these levels
```

<!-- DOCGEN:OVERVIEW:END -->

<!-- DOCGEN:TOC:START -->

See the [documentation](docs/) for detailed usage instructions:
- [Overview](docs/01-overview.md)
- [Configuration](docs/04-configuration.md)

<!-- DOCGEN:TOC:END -->
