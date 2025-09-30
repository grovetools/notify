Generate an overview section for grove-notifications.

## Requirements
Create a comprehensive overview that includes:

1. **High-level description**: What grove-notifications is and its purpose in the Grove ecosystem
2. **Animated GIF placeholder**: Include `<!-- placeholder for animated gif -->`
3. **Key features**: System notifications, ntfy integration, programmatic notification support, Grove ecosystem integration
4. **Ecosystem Integration**: Explain how grove-notifications fits with other Grove tools for alerting and monitoring
5. **How it works**: Provide a more technical description and exactly what happens under the hood
6. **Installation**: Include brief installation instructions at the bottom

## Installation Format
Include this condensed installation section at the bottom:

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

## Context
Grove-notifications provides a unified notification system for the Grove ecosystem, supporting both system notifications and remote notifications via ntfy. It enables developers to receive alerts about build status, test results, deployment events, and other important events in their development workflow.