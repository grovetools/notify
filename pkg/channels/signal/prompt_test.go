package signal

import "strings"

import "testing"

// Every Signal-clawed agent is briefed with these instructions, so this is
// where the channel-provenance action policy has to live to cover all of them
// — not just the standing assistant (assistant-pane spec §3.7).
//
// Inbound text reaches a permissioned agent's stdin behind nothing but a
// phone-number allowlist. That is the right bar for asking for work and the
// wrong one for destroying things, and the tag the routing layer stamps on
// every inbound line is what lets the agent tell the two apart.
func TestDefaultAgentInstructionsCarryTheProvenancePolicy(t *testing.T) {
	instr := DefaultAgentInstructions("household")

	if !strings.Contains(instr, "[via Signal from <name>]") {
		t.Errorf("instructions never show the provenance tag:\n%s", instr)
	}
	for _, want := range []string{
		"provenance tag is load-bearing",
		"deleting plans, worktrees or branches",
		"--force operation",
		"untagged confirmation",
	} {
		if !strings.Contains(instr, want) {
			t.Errorf("instructions are missing %q from the destructive-action policy:\n%s", want, instr)
		}
	}
}

// The notice is unconditional: an agent with no configured target is reachable
// by exactly the same inbound path, so it needs exactly the same rule.
func TestProvenancePolicyDoesNotDependOnATarget(t *testing.T) {
	if !strings.Contains(DefaultAgentInstructions(""), ChannelProvenanceNotice) {
		t.Error("a target-less agent was briefed without the provenance policy")
	}
}
