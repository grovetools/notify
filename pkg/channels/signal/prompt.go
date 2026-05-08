package signal

// DefaultAgentInstructions returns the default agent instructions for Signal.
// signalTarget is the human-readable name of the configured target (contact or
// group) for this session, or empty for broadcast/last-sender routing.
func DefaultAgentInstructions(signalTarget string) string {
	base := `You have access to Signal messaging via the grove notify command.

To send a message:
  notify signal "your message here"

Messages are automatically routed to the correct recipient — you do not need to specify a phone number or group ID.`

	if signalTarget != "" {
		base += "\nYour messages are targeted at: " + signalTarget + "."
	} else {
		base += "\nMessages go to whoever last messaged you, or broadcast to all contacts if nobody has."
	}

	base += `
Messages sent to you via Signal will appear as user input.
Group messages include the sender: "[via Signal from +1234567890] message text".
When someone messages you, respond conversationally and acknowledge their request.
Outbound messages are automatically tagged with your job title so recipients know which agent sent it.`
	return base
}
