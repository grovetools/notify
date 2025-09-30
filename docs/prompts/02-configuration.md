Generate a configuration section for grove-notifications.

## Requirements
Create a comprehensive configuration guide that includes:

1. **Configuration Overview**: Explain how grove-notifications is configured
2. **Notification Types**: Document the different notification backends (system, ntfy)
3. **Configuration Methods**: CLI flags, environment variables, and configuration files
4. **System Notifications**: Configuration for native system notifications
5. **Ntfy Integration**: Configuration for ntfy.sh remote notifications including URL and topic setup
6. **Priority Levels**: Available priority/urgency levels for notifications
7. **Tags and Categories**: How to use tags for organizing notifications
8. **Examples**: Practical configuration examples for common use cases

## Context
Grove-notifications supports multiple notification backends and can be configured through various methods. The primary backends are system notifications (native OS notifications) and ntfy (remote push notifications). Configuration should be flexible to support different development workflows and notification preferences.

## Technical Details
Based on the code structure:
- NotificationType: "system" or "ntfy"
- Config struct includes Type, NtfyURL, NtfyTopic
- Priority levels and tags are supported
- Convenience functions available for both notification types