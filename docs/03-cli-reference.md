# CLI Reference

Complete command reference for `notify`.

## notify

<div class="terminal">
<span class="term-bold term-fg-11">NOTIFY</span>
 <span class="term-italic">Notification system for Grove ecosystem</span>

 <span class="term-italic term-fg-11">USAGE</span>
 notify [command]

 <span class="term-italic term-fg-11">COMMANDS</span>
 <span class="term-bold term-fg-4">completion</span>  Generate the autocompletion script for the specified shell
 <span class="term-bold term-fg-4">ntfy</span>        Send a notification via ntfy
 <span class="term-bold term-fg-4">system</span>      Send a system notification
 <span class="term-bold term-fg-4">version</span>     Print the version information for this binary

 <span class="term-dim">Flags: -c/--config, -h/--help, --json, -v/--verbose</span>

 Use "notify [command] --help" for more information.
</div>

### notify ntfy

<div class="terminal">
<span class="term-bold term-fg-11">NOTIFY NTFY</span>
 <span class="term-italic">Send a notification via ntfy</span>

 Send a notification via ntfy service.
 Usage:
 notify ntfy --topic mytopic --title "Title" "Message text"
 notify ntfy --topic mytopic --title "Title" --priority
 high --tags tag1,tag2 "Message text"

 <span class="term-italic term-fg-11">USAGE</span>
 notify ntfy [args...] [flags]

 <span class="term-italic term-fg-11">FLAGS</span>
 <span class="term-fg-5">-h, --help</span>      help for ntfy
 <span class="term-fg-5">    --priority</span>  Priority (min, low, default, high, urgent)<span class="term-dim"> (default: default)</span>
 <span class="term-fg-5">    --tags</span>      Comma-separated tags
 <span class="term-fg-5">    --title</span>     Notification title<span class="term-dim"> (default: Grove Notification)</span>
 <span class="term-fg-5">    --topic</span>     Ntfy topic (required)
 <span class="term-fg-5">    --url</span>       Ntfy server URL (default: https://ntfy.sh)
</div>

### notify system

<div class="terminal">
<span class="term-bold term-fg-11">NOTIFY SYSTEM</span>
 <span class="term-italic">Send a system notification</span>

 Send a system notification with title and message.
 Usage:
   notify system --title "Title" "Message text"
 notify system --title "Title" --level info "Message text"

 <span class="term-italic term-fg-11">USAGE</span>
 notify system [args...] [flags]

 <span class="term-italic term-fg-11">FLAGS</span>
 <span class="term-fg-5">-h, --help</span>   help for system
 <span class="term-fg-5">    --level</span>  Notification level (info, warning, error)<span class="term-dim"> (default: info)</span>
 <span class="term-fg-5">    --title</span>  Notification title<span class="term-dim"> (default: Grove Notification)</span>
</div>

### notify version

<div class="terminal">
<span class="term-bold term-fg-11">NOTIFY VERSION</span>
 <span class="term-italic">Print the version information for this binary</span>

 <span class="term-italic term-fg-11">USAGE</span>
 notify version [flags]

 <span class="term-italic term-fg-11">FLAGS</span>
 <span class="term-fg-5">-h, --help</span>  help for version
 <span class="term-fg-5">    --json</span>  Output version information in JSON format
</div>

