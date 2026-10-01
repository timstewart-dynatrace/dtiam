package get

import "testing"

func TestBuildAuditFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		eventType string
		user      string
		want      string
	}{
		{name: "should be empty with no flags", want: ""},
		{
			name: "should build an event type clause", eventType: "DELETE",
			want: `eventType = "DELETE"`,
		},
		{
			name: "should build a user clause", user: "alice@example.com",
			want: `user = "alice@example.com"`,
		},
		{
			name: "should and the clauses together", eventType: "CREATE", user: "bob@example.com",
			want: `eventType = "CREATE" and user = "bob@example.com"`,
		},
		{
			// An explicit filter must win, so users are never limited by the
			// convenience flags' coverage.
			name:   "should let an explicit filter override the flags",
			filter: `resource = "GROUP"`, eventType: "DELETE", user: "alice@example.com",
			want: `resource = "GROUP"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditFilterFlag = tt.filter
			auditEventTypeFlag = tt.eventType
			auditUserFlag = tt.user
			t.Cleanup(func() {
				auditFilterFlag, auditEventTypeFlag, auditUserFlag = "", "", ""
			})

			if got := buildAuditFilter(); got != tt.want {
				t.Errorf("buildAuditFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuditLogsCommandMetadata(t *testing.T) {
	if auditLogsCmd.Short == "" {
		t.Error("Short is required")
	}
	if auditLogsCmd.Long == "" {
		t.Error("Long is required")
	}
	if auditLogsCmd.Example == "" {
		t.Error("Example is required")
	}
	if auditLogsCmd.RunE == nil {
		t.Error("commands must use RunE, not Run")
	}

	// Aliases keep the kubectl-style singular/plural forms working.
	wantAlias := map[string]bool{"audit-log": false, "audits": false, "audit": false}
	for _, a := range auditLogsCmd.Aliases {
		wantAlias[a] = true
	}
	for alias, found := range wantAlias {
		if !found {
			t.Errorf("missing alias %q", alias)
		}
	}
}
