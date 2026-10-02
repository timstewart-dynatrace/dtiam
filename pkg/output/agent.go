package output

import (
	"strings"
	"sync"
)

// AgentEnvelope is the single JSON document dtiam writes to stdout in agent
// mode (--agent / -A). ok is false only when the command failed; a command
// that reports a non-zero result (diff finding drift, can-i answering no) is
// ok with exit_code set in context.
type AgentEnvelope struct {
	OK      bool          `json:"ok"`
	Result  any           `json:"result"`
	Error   *AgentError   `json:"error,omitempty"`
	Context *AgentContext `json:"context"`
}

// AgentError is a machine-readable failure.
type AgentError struct {
	// Code is one of: safety_blocked, usage, bad_request, auth_required,
	// permission_denied, not_found, conflict, rate_limited, server_error, error.
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	StatusCode  int      `json:"status_code,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// AgentContext carries metadata about the run alongside the result.
type AgentContext struct {
	Command     string   `json:"command"`
	Operation   string   `json:"operation,omitempty"`
	SafetyLevel string   `json:"safety_level,omitempty"`
	DryRun      bool     `json:"dry_run,omitempty"`
	Total       *int     `json:"total,omitempty"`
	ExitCode    int      `json:"exit_code"`
	Duration    string   `json:"duration"`
	Messages    []string `json:"messages,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Version     string   `json:"version"`
}

// AgentSession collects what a command would have printed, for the envelope.
type AgentSession struct {
	mu       sync.Mutex
	results  []any
	messages []string
	warnings []string
}

// NewAgentSession creates an empty session.
func NewAgentSession() *AgentSession { return &AgentSession{} }

func (s *AgentSession) addResult(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, v)
}

// AddMessage records an informational line.
func (s *AgentSession) AddMessage(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

func (s *AgentSession) addWarning(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warnings = append(s.warnings, strings.TrimSpace(msg))
}

// Result returns the collected result: nil for none, the value for one, and a
// list for several (a command that printed more than one data set).
func (s *AgentSession) Result() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch len(s.results) {
	case 0:
		return nil
	case 1:
		return s.results[0]
	default:
		return append([]any(nil), s.results...)
	}
}

// Total returns the item count when the result is a single list.
func (s *AgentSession) Total() *int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.results) != 1 {
		return nil
	}
	var n int
	switch v := s.results[0].(type) {
	case []map[string]any:
		n = len(v)
	case []any:
		n = len(v)
	default:
		return nil
	}
	return &n
}

// Messages returns the collected informational lines.
func (s *AgentSession) Messages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...)
}

// Warnings returns the collected warnings.
func (s *AgentSession) Warnings() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.warnings...)
}
