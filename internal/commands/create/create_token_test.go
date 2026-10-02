package create

import (
	"strings"
	"testing"
)

func TestTokenResources(t *testing.T) {
	tests := []struct {
		name     string
		explicit []string
		envs     []string
		want     string
	}{
		{name: "should default to the account", want: "urn:dtaccount:acct"},
		{name: "should map environments to URNs", envs: []string{"e1", "e2"}, want: "urn:dtenvironment:e1|urn:dtenvironment:e2"},
		{name: "should keep explicit URNs first", explicit: []string{"urn:x"}, envs: []string{"e1"}, want: "urn:x|urn:dtenvironment:e1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Join(tokenResources(tt.explicit, tt.envs, "acct"), "|"); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateTokenCmd_RequiredFlags(t *testing.T) {
	for _, name := range []string{"name", "scopes", "user"} {
		f := tokenCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("--%s is missing", name)
		}
		if f.Annotations["cobra_annotation_bash_completion_one_required_flag"] == nil {
			t.Errorf("--%s should be marked required", name)
		}
	}
	for _, name := range []string{"expires-in", "expires-at", "environment", "resource", "tag"} {
		if tokenCmd.Flags().Lookup(name) == nil {
			t.Errorf("--%s is missing", name)
		}
	}
}
