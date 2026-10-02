package account

import (
	"strings"
	"testing"
)

func TestPickDefaultSubscription(t *testing.T) {
	sub := func(uuid, status string) map[string]any {
		return map[string]any{"uuid": uuid, "status": status}
	}

	tests := []struct {
		name    string
		subs    []map[string]any
		want    string
		wantErr string
	}{
		{name: "should error when there are no subscriptions", wantErr: "no subscriptions"},
		{name: "should use the only subscription whatever its status", subs: []map[string]any{sub("s1", "EXPIRED")}, want: "s1"},
		{
			// The shape of a real account: expired, current and pending terms.
			name: "should pick the single ACTIVE subscription among several",
			subs: []map[string]any{sub("old", "EXPIRED"), sub("now", "ACTIVE"), sub("next", "PENDING")},
			want: "now",
		},
		{
			name:    "should ask for a choice when none is ACTIVE",
			subs:    []map[string]any{sub("a", "EXPIRED"), sub("b", "PENDING")},
			wantErr: "none is ACTIVE",
		},
		{
			name:    "should ask for a choice when several are ACTIVE",
			subs:    []map[string]any{sub("a", "ACTIVE"), sub("b", "active")},
			wantErr: "2 ACTIVE subscriptions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickDefaultSubscription(tt.subs)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
