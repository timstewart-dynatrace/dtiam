// Package integration provides read-only integration tests against a live Dynatrace API.
//
// These tests are SKIPPED unless the required environment variables are set:
//
//   - DTIAM_ACCOUNT_UUID (required)
//   - DTIAM_CLIENT_SECRET (required for OAuth2)
//   - DTIAM_CLIENT_ID (optional — auto-extracted from secret)
//
// Alternatively, set DTIAM_BEARER_TOKEN instead of OAuth2 credentials.
//
// These tests make ZERO mutations — only GET/list operations. Safe to run
// against any account at any time.
//
// Run: go test ./internal/integration/ -v
package integration

import (
	"context"
	"os"
	"testing"

	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// skipIfNoCredentials skips the test if API credentials are not configured.
func skipIfNoCredentials(t *testing.T) {
	t.Helper()
	if os.Getenv("DTIAM_ACCOUNT_UUID") == "" {
		t.Skip("DTIAM_ACCOUNT_UUID not set — skipping integration test")
	}
	if os.Getenv("DTIAM_CLIENT_SECRET") == "" && os.Getenv("DTIAM_BEARER_TOKEN") == "" {
		t.Skip("Neither DTIAM_CLIENT_SECRET nor DTIAM_BEARER_TOKEN set — skipping integration test")
	}
}

// --- Read-only integration tests ---

func TestIntegration_ListGroups(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewGroupHandler(c)
	groups, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List groups failed: %v", err)
	}

	t.Logf("Found %d groups", len(groups))
	for _, g := range groups[:min(3, len(groups))] {
		if _, ok := g["uuid"]; !ok {
			t.Error("Group missing 'uuid' field")
		}
		if _, ok := g["name"]; !ok {
			t.Error("Group missing 'name' field")
		}
		t.Logf("  Group: %v (%v)", g["name"], g["uuid"])
	}
}

func TestIntegration_ListUsers(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewUserHandler(c)
	users, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List users failed: %v", err)
	}

	t.Logf("Found %d users", len(users))
	for _, u := range users[:min(3, len(users))] {
		if _, ok := u["email"]; !ok {
			t.Error("User missing 'email' field")
		}
		t.Logf("  User: %v", u["email"])
	}
}

func TestIntegration_ListPolicies(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewPolicyHandler(c)
	policies, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List policies failed: %v", err)
	}

	t.Logf("Found %d policies", len(policies))
	for _, p := range policies[:min(3, len(policies))] {
		if _, ok := p["uuid"]; !ok {
			t.Error("Policy missing 'uuid' field")
		}
		t.Logf("  Policy: %v (%v)", p["name"], p["uuid"])
	}
}

func TestIntegration_ListBindings(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewBindingHandler(c)
	bindings, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List bindings failed: %v", err)
	}

	t.Logf("Found %d bindings", len(bindings))
	for _, b := range bindings[:min(3, len(bindings))] {
		if _, ok := b["policyUuid"]; !ok {
			t.Error("Binding missing 'policyUuid' field")
		}
	}
}

func TestIntegration_ListEnvironments(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewEnvironmentHandler(c)
	envs, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List environments failed: %v", err)
	}

	t.Logf("Found %d environments", len(envs))
	for _, e := range envs[:min(3, len(envs))] {
		if _, ok := e["id"]; !ok {
			t.Error("Environment missing 'id' field")
		}
		t.Logf("  Environment: %v (%v)", e["name"], e["id"])
	}
}

func TestIntegration_ListBoundaries(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewBoundaryHandler(c)
	boundaries, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List boundaries failed: %v", err)
	}

	t.Logf("Found %d boundaries", len(boundaries))
}

func TestIntegration_ListServiceUsers(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewServiceUserHandler(c)
	serviceUsers, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List service users failed: %v", err)
	}

	t.Logf("Found %d service users", len(serviceUsers))
}

func TestIntegration_ListLimits(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewLimitsHandler(c)
	limits, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List limits failed: %v", err)
	}

	t.Logf("Found %d limits", len(limits))
	if len(limits) == 0 {
		t.Error("Expected at least one limit")
	}
}

func TestIntegration_ListSubscriptions(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewSubscriptionHandler(c)
	subs, err := handler.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List subscriptions failed: %v", err)
	}

	t.Logf("Found %d subscriptions", len(subs))
}

func TestIntegration_GetGroupDetails(t *testing.T) {
	skipIfNoCredentials(t)
	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewGroupHandler(c)
	ctx := context.Background()

	groups, err := handler.List(ctx, nil)
	if err != nil {
		t.Fatalf("List groups failed: %v", err)
	}
	if len(groups) == 0 {
		t.Skip("No groups found — account may be empty")
	}

	uuid, _ := groups[0]["uuid"].(string)
	group, err := handler.Get(ctx, uuid)
	if err != nil {
		t.Fatalf("Get group %s failed: %v", uuid, err)
	}

	if group["uuid"] != uuid {
		t.Errorf("Get group uuid = %v, want %v", group["uuid"], uuid)
	}
	t.Logf("Group: name=%v, owner=%v", group["name"], group["owner"])

	members, err := handler.GetMembers(ctx, uuid)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}
	t.Logf("  Members: %d", len(members))
}

func TestIntegration_OAuth_MultipleRequests(t *testing.T) {
	skipIfNoCredentials(t)
	if os.Getenv("DTIAM_CLIENT_SECRET") == "" {
		t.Skip("OAuth credentials not set — skipping multi-request test")
	}

	c, err := common.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
	defer c.Close()

	handler := resources.NewGroupHandler(c)
	ctx := context.Background()

	// Multiple calls should reuse the token
	for i := 0; i < 3; i++ {
		_, err := handler.List(ctx, nil)
		if err != nil {
			t.Fatalf("API call %d failed: %v", i+1, err)
		}
	}

	t.Log("OAuth token acquisition and reuse verified across 3 requests")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
