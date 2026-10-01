package cli

import (
	"os"
	"strings"
)

// AgentInfo describes a detected non-interactive caller.
type AgentInfo struct {
	// Detected is true when dtiam appears to be running without a human at the
	// terminal.
	Detected bool

	// Name identifies what was detected, for the User-Agent and verbose output.
	Name string

	// Source is the environment variable that triggered detection, so the
	// decision is explainable rather than mysterious.
	Source string
}

// knownAgents maps an environment variable to the agent it indicates.
//
// These are coding agents that run dtiam on a user's behalf with no terminal to
// prompt at. Detecting them lets dtiam default to machine-readable output instead
// of emitting ANSI colors into a transcript and blocking on a confirmation
// prompt nobody can answer.
var knownAgents = map[string]string{
	"CLAUDECODE":     "claude-code",
	"CLAUDE_CODE":    "claude-code",
	"CURSOR_AGENT":   "cursor",
	"GITHUB_COPILOT": "github-copilot",
	"CODEIUM_AGENT":  "codeium",
	"TABNINE_AGENT":  "tabnine",
	"AMAZON_Q":       "amazon-q",
	"JUNIE":          "junie",
	"KIRO":           "kiro",
	"OPENCODE":       "opencode",
	"AIDER":          "aider",
	"AI_AGENT":       "generic-ai",
}

// knownCI maps an environment variable to the CI system it indicates.
//
// CI is treated separately from agents: both are non-interactive, but CI output
// is usually read by a human later in a log, so only colors and prompts are
// unwanted — not necessarily the table format.
var knownCI = map[string]string{
	"GITHUB_ACTIONS":         "github-actions",
	"GITLAB_CI":              "gitlab-ci",
	"JENKINS_URL":            "jenkins",
	"CIRCLECI":               "circleci",
	"TRAVIS":                 "travis",
	"BUILDKITE":              "buildkite",
	"TEAMCITY_VERSION":       "teamcity",
	"TF_BUILD":               "azure-pipelines",
	"BITBUCKET_BUILD_NUMBER": "bitbucket",
	"CI":                     "generic-ci",
}

// EnvDisableAgentDetection turns detection off, for anyone who wants dtiam's
// interactive behavior inside an agent or CI session.
const EnvDisableAgentDetection = "DTIAM_NO_AGENT_DETECT"

// truthy reports whether an environment variable value should count as set.
// "0" and "false" are treated as unset, since some tools export their flag
// unconditionally and switch it with the value.
func truthy(value string) bool {
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

// DetectAgent reports whether dtiam is running under a coding agent.
func DetectAgent() AgentInfo {
	if truthy(os.Getenv(EnvDisableAgentDetection)) {
		return AgentInfo{}
	}
	return detect(knownAgents)
}

// DetectCI reports whether dtiam is running in a CI system.
func DetectCI() AgentInfo {
	if truthy(os.Getenv(EnvDisableAgentDetection)) {
		return AgentInfo{}
	}
	return detect(knownCI)
}

// genericNames are fallback labels that carry no information about which tool is
// running. A specific match always wins over these, so a host with both
// CLAUDECODE and AI_AGENT set reports claude-code rather than generic-ai.
var genericNames = map[string]bool{"generic-ai": true, "generic-ci": true}

// detect scans a variable-to-name map, preferring specific matches.
//
// Map iteration order is random in Go, so a host with several variables set would
// otherwise report a different agent on each run. Keys are sorted to make the
// choice reproducible, and generic labels are held back as a fallback.
func detect(candidates map[string]string) AgentInfo {
	keys := make([]string, 0, len(candidates))
	for k := range candidates {
		keys = append(keys, k)
	}
	sortStrings(keys)

	var fallback AgentInfo
	for _, envVar := range keys {
		if !truthy(os.Getenv(envVar)) {
			continue
		}
		info := AgentInfo{Detected: true, Name: candidates[envVar], Source: envVar}
		if genericNames[info.Name] {
			if !fallback.Detected {
				fallback = info
			}
			continue
		}
		return info
	}
	return fallback
}

// sortStrings sorts in place. Kept local to avoid pulling "sort" into a package
// that is imported by every command.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// UserAgentSuffix returns a User-Agent suffix naming the detected caller, or "".
func UserAgentSuffix() string {
	if info := DetectAgent(); info.Detected {
		return " (agent: " + info.Name + ")"
	}
	if info := DetectCI(); info.Detected {
		return " (ci: " + info.Name + ")"
	}
	return ""
}
