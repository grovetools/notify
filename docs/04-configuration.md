## Notification Settings

This section outlines the configuration options available for the `notifications` extension. This extension allows the Grove ecosystem to dispatch alerts through various channels, such as the ntfy.sh service or native system notifications.

| Property | Description |
| :--- | :--- |
| `ntfy` | (object, optional) <br>Configuration settings for the **ntfy** notification service. This allows you to send push notifications to mobile or desktop clients via topics. |
| `system` | (object, optional) <br>Configuration settings for native desktop notifications (e.g., macOS Notification Center, Linux `notify-send`). |

```toml
[notifications.ntfy]
enabled = true
topic = "my-team-alerts"

[notifications.system]
levels = ["error", "warning"]
```

### Ntfy Configuration

These settings control the integration with the [ntfy.sh](https://ntfy.sh) service (or a self-hosted instance).

| Property | Description |
| :--- | :--- |
| `enabled` | (boolean, required) <br>Master switch for the ntfy integration. When set to `true`, notifications will be dispatched to the configured server. |
| `topic` | (string, required) <br>The specific topic name to publish notifications to. This serves as the channel identifier; any client subscribed to this topic will receive the alerts. |
| `url` | (string, required) <br>The base URL of the ntfy server. Use `https://ntfy.sh` for the public service, or provide the URL of your self-hosted instance. |

### System Configuration

These settings control how Grove tools interact with your operating system's native notification center.

| Property | Description |
| :--- | :--- |
| `levels` | (array of strings, required) <br>A list of severity levels that should trigger a desktop popup. Common values are "info", "warning", and "error". This allows you to filter out noise and only be alerted for significant events (e.g., only showing "error" messages). |