package output

import "testing"

func TestAgentPrinterCollectsInsteadOfWriting(t *testing.T) {
	s := NewAgentSession()
	p := NewAgentPrinter(s)

	if err := p.Print([]map[string]any{{"a": 1}, {"a": 2}}, nil); err != nil {
		t.Fatal(err)
	}
	p.PrintSuccess("created %s", "x")
	p.PrintWarning("careful")
	p.PrintMessage("  ")

	if got := s.Total(); got == nil || *got != 2 {
		t.Errorf("Total() = %v, want 2", got)
	}
	if msgs := s.Messages(); len(msgs) != 1 || msgs[0] != "created x" {
		t.Errorf("Messages() = %v (blank lines must be dropped)", msgs)
	}
	if w := s.Warnings(); len(w) != 1 || w[0] != "careful" {
		t.Errorf("Warnings() = %v", w)
	}
}

func TestAgentSessionResultShape(t *testing.T) {
	s := NewAgentSession()
	if s.Result() != nil {
		t.Error("no results should be nil")
	}
	s.addResult(map[string]any{"x": 1})
	if _, ok := s.Result().(map[string]any); !ok {
		t.Error("one result should be returned as itself")
	}
	s.addResult("second")
	if r, ok := s.Result().([]any); !ok || len(r) != 2 {
		t.Errorf("several results should be a list, got %T", s.Result())
	}
	if s.Total() != nil {
		t.Error("Total() only applies to a single list result")
	}
}
