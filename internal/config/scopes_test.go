package config

import "testing"

func TestGetEffectiveScopes(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		cred    *Credential
		want    string
		comment string
	}{
		{
			name: "should return empty when nothing is set, meaning use defaults",
			want: "",
		},
		{
			name:   "should prefer the environment variable",
			envVar: "scope-from-env",
			cred:   &Credential{Scopes: "scope-from-cred"},
			want:   "scope-from-env",
		},
		{
			name: "should fall back to the credential",
			cred: &Credential{Scopes: "scope-from-cred"},
			want: "scope-from-cred",
		},
		{
			name: "should ignore an empty credential scope",
			cred: &Credential{Scopes: ""},
			want: "",
		},
		{
			name: "should tolerate a nil credential",
			cred: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				t.Setenv(EnvScopes, tt.envVar)
			} else {
				t.Setenv(EnvScopes, "")
			}

			if got := GetEffectiveScopes(tt.cred); got != tt.want {
				t.Errorf("GetEffectiveScopes() = %q, want %q", got, tt.want)
			}
		})
	}
}
