package notifiers

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SystemNotifier struct{}

func NewSystemNotifier() *SystemNotifier {
	return &SystemNotifier{}
}

func (n *SystemNotifier) Send(title, message, level string) error {
	// Use golang.org/x/text/cases instead of deprecated strings.Title
	caser := cases.Title(language.English)
	fullTitle := fmt.Sprintf("%s: %s", title, caser.String(level))

	switch runtime.GOOS {
	case "darwin":
		// macOS notification using osascript
		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`,
			escapeForAppleScript(message), escapeForAppleScript(fullTitle))
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()
	case "linux":
		// Linux notification using notify-send
		cmd := exec.Command("notify-send", fullTitle, message)
		return cmd.Run()
	default:
		return fmt.Errorf("system notifications not supported on %s", runtime.GOOS)
	}
}

func escapeForAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}