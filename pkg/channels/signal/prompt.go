package signal

// DefaultAgentInstructions returns the default agent instructions for Signal,
// parameterized with the account number from config.
func DefaultAgentInstructions(_ string) string {
	return `You have access to Signal messaging via the grove notify command.

To send a message (replies to the last person who messaged you, or broadcasts to all contacts):
  notify signal "your message here"

To send to a specific contact:
  notify signal --to "+1234567890" "your message"

Messages sent to you via Signal will appear as user input.
When someone messages you, respond conversationally and acknowledge their request.
Outbound messages are automatically tagged with your job title so recipients know which agent sent it.`
}
