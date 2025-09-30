`grove-notifications` is a command-line tool and Go package for sending local system notifications and remote push notifications via ntfy.sh.

<!-- placeholder for animated gif -->

### Key Features

*   Sends native desktop notifications on macOS (via `osascript`) and Linux (via `notify-send`).
*   Sends remote push notifications to a specified topic on an ntfy server.
*   Provides a Go API (`SendSystem`, `SendNtfy`) for programmatic use within other applications.
*   Can be executed by other Grove tools to signal outcomes of operations like builds or tests.

### Ecosystem Integration

The `notify` binary is designed to be called by other processes. For example, a build script or another Grove tool can execute `notify system "Build complete"` to alert the user when a long-running task finishes. This allows for event-driven feedback within the local development environment.

### How It Works

The tool is a Go application that provides two primary notification mechanisms:

*   **System Notifications**: The `system` subcommand checks the host operating system. On macOS, it executes `osascript` with a script to display a notification. On Linux, it executes the `notify-send` command.
*   **Ntfy Notifications**: The `ntfy` subcommand constructs and sends an HTTP POST request to a specified ntfy server URL. The message content is sent as the request body, while metadata such as title, priority, and tags are passed as HTTP headers.

### Installation

Install via the Grove meta-CLI:
```bash
grove install notifications
```

Verify installation:
```bash
notify version
```

Requires the `grove` meta-CLI. See the [Grove Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md) if you don't have it installed.