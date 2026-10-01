package cli

import "testing"

func TestTruthy(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty is unset", value: "", want: false},
		{name: "1 is set", value: "1", want: true},
		{name: "true is set", value: "true", want: true},
		{name: "arbitrary value is set", value: "claude", want: true},
		// Some tools export their flag unconditionally and switch it by value,
		// so these must not count as set.
		{name: "0 is unset", value: "0", want: false},
		{name: "false is unset", value: "false", want: false},
		{name: "FALSE is unset regardless of case", value: "FALSE", want: false},
		{name: "no is unset", value: "no", want: false},
		{name: "off is unset", value: "off", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truthy(tt.value); got != tt.want {
				t.Errorf("truthy(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// clearDetectionEnv unsets every variable detection looks at, so a test is not
// influenced by the environment it happens to run in -- including the agent that
// may be running the test suite itself.
func clearDetectionEnv(t *testing.T) {
	t.Helper()
	for k := range knownAgents {
		t.Setenv(k, "")
	}
	for k := range knownCI {
		t.Setenv(k, "")
	}
	t.Setenv(EnvDisableAgentDetection, "")
}

func TestDetectAgent_NothingSet(t *testing.T) {
	clearDetectionEnv(t)
	if got := DetectAgent(); got.Detected {
		t.Errorf("Detected = true with no agent variables set (%+v)", got)
	}
}

func TestDetectAgent_IdentifiesAgentAndSource(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("CLAUDECODE", "1")

	got := DetectAgent()
	if !got.Detected {
		t.Fatal("Detected = false with CLAUDECODE set")
	}
	if got.Name != "claude-code" {
		t.Errorf("Name = %q, want claude-code", got.Name)
	}
	// Source makes the decision explainable rather than mysterious.
	if got.Source != "CLAUDECODE" {
		t.Errorf("Source = %q, want CLAUDECODE", got.Source)
	}
}

func TestDetectAgent_SpecificAgentBeatsGenericFallback(t *testing.T) {
	// A host with both set should report the informative name, not "generic-ai".
	clearDetectionEnv(t)
	t.Setenv("AI_AGENT", "1")
	t.Setenv("CLAUDECODE", "1")

	if got := DetectAgent(); got.Name != "claude-code" {
		t.Errorf("Name = %q, want claude-code to win over generic-ai", got.Name)
	}
}

func TestDetectAgent_GenericFallbackStillWorksAlone(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("AI_AGENT", "1")

	got := DetectAgent()
	if !got.Detected || got.Name != "generic-ai" {
		t.Errorf("got %+v, want generic-ai detected", got)
	}
}

func TestDetectAgent_IsDeterministicAcrossRuns(t *testing.T) {
	// Map iteration order is random in Go; without sorting, a host with several
	// variables set would report a different agent each run.
	clearDetectionEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CURSOR_AGENT", "1")
	t.Setenv("AIDER", "1")

	first := DetectAgent().Name
	for i := 0; i < 50; i++ {
		if got := DetectAgent().Name; got != first {
			t.Fatalf("detection is not deterministic: got %q then %q", first, got)
		}
	}
}

func TestDetectAgent_RespectsOptOut(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv(EnvDisableAgentDetection, "1")

	if got := DetectAgent(); got.Detected {
		t.Errorf("Detected = true despite %s being set", EnvDisableAgentDetection)
	}
}

func TestDetectAgent_IgnoresFalsyValues(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("CLAUDECODE", "0")

	if got := DetectAgent(); got.Detected {
		t.Error("Detected = true for CLAUDECODE=0")
	}
}

func TestDetectCI(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("GITHUB_ACTIONS", "true")

	got := DetectCI()
	if !got.Detected || got.Name != "github-actions" {
		t.Errorf("got %+v, want github-actions detected", got)
	}
}

func TestDetectCI_SpecificBeatsGenericCI(t *testing.T) {
	clearDetectionEnv(t)
	t.Setenv("CI", "true")
	t.Setenv("GITLAB_CI", "true")

	if got := DetectCI(); got.Name != "gitlab-ci" {
		t.Errorf("Name = %q, want gitlab-ci to win over generic-ci", got.Name)
	}
}

func TestDetectCI_DoesNotReportAgents(t *testing.T) {
	// The two detections are separate; an agent must not register as CI.
	clearDetectionEnv(t)
	t.Setenv("CLAUDECODE", "1")

	if got := DetectCI(); got.Detected {
		t.Errorf("DetectCI() reported %+v for an agent variable", got)
	}
}

func TestUserAgentSuffix(t *testing.T) {
	clearDetectionEnv(t)
	if got := UserAgentSuffix(); got != "" {
		t.Errorf("UserAgentSuffix() = %q, want empty with nothing detected", got)
	}

	t.Setenv("CLAUDECODE", "1")
	if got := UserAgentSuffix(); got != " (agent: claude-code)" {
		t.Errorf("UserAgentSuffix() = %q", got)
	}
}

func TestUserAgentSuffix_PrefersAgentOverCI(t *testing.T) {
	// An agent running inside CI is better described as the agent.
	clearDetectionEnv(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("CLAUDECODE", "1")

	if got := UserAgentSuffix(); got != " (agent: claude-code)" {
		t.Errorf("UserAgentSuffix() = %q, want the agent to win", got)
	}
}

func TestStateIsAgent(t *testing.T) {
	s := &State{}
	if s.IsAgent() {
		t.Error("IsAgent() = true with no agent name set")
	}
	s.AgentName = "claude-code"
	if !s.IsAgent() {
		t.Error("IsAgent() = false with an agent name set")
	}
}

func TestSortStrings(t *testing.T) {
	got := []string{"c", "a", "b", "a"}
	sortStrings(got)
	want := []string{"a", "a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortStrings() = %v, want %v", got, want)
		}
	}
}
