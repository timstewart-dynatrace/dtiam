# API Reference

> **DISCLAIMER:** This tool is provided "as-is" without warranty. Use at your own risk. This is an independent, community-developed tool and is **NOT produced, endorsed, or supported by Dynatrace**.

Programmatic usage guide for dtiam Go packages.

## Overview

While dtiam is primarily a CLI tool, its internal packages can be imported and used programmatically for scripting and automation in Go applications.

## Package Import

```go
import (
    "github.com/jtimothystewart/dtiam/internal/config"
    "github.com/jtimothystewart/dtiam/internal/client"
    "github.com/jtimothystewart/dtiam/internal/auth"
    "github.com/jtimothystewart/dtiam/internal/resources"
    "github.com/jtimothystewart/dtiam/internal/output"
)
```

## Configuration

### Loading Configuration

```go
import "github.com/jtimothystewart/dtiam/internal/config"

// Load existing configuration from ~/.config/dtiam/config
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Access current context
ctx := cfg.GetCurrentContext()
fmt.Printf("Account UUID: %s\n", ctx.Context.AccountUUID)

// Get specific context
prodCtx := cfg.GetContext("production")

// Get credential
cred := cfg.GetCredential("prod-creds")
fmt.Printf("Client ID: %s\n", cred.Credential.ClientID)
```

### Creating Configuration Programmatically

```go
import "github.com/jtimothystewart/dtiam/internal/config"

cfg := config.NewConfig()

// Add credentials
cfg.SetCredential("prod-creds", "dt0s01.XXXX", "dt0s01.XXXX.YYYY")

// Add context
cfg.SetContext("production", "abc-123-def", "prod-creds")

// Set as current context
cfg.SetCurrentContext("production")

// Save to file
if err := config.Save(cfg); err != nil {
    log.Fatal(err)
}
```

### Environment Variables

```go
import (
    "os"
    "github.com/jtimothystewart/dtiam/internal/config"
)

// Check for environment overrides
accountUUID := os.Getenv("DTIAM_ACCOUNT_UUID")
clientID := os.Getenv("DTIAM_CLIENT_ID")
clientSecret := os.Getenv("DTIAM_CLIENT_SECRET")
bearerToken := os.Getenv("DTIAM_BEARER_TOKEN")
```

## HTTP Client

### Creating a Client

```go
import (
    "github.com/jtimothystewart/dtiam/internal/client"
    "github.com/jtimothystewart/dtiam/internal/auth"
)

// Create OAuth2 token manager
tokenMgr := auth.NewOAuthTokenManager(
    "dt0s01.XXXX",           // client ID
    "dt0s01.XXXX.YYYY",      // client secret
    "abc-123-def",           // account UUID
)

// Create client
c := client.New(
    "abc-123-def",           // account UUID
    tokenMgr,                // token provider
    client.WithVerbose(true),
    client.WithTimeout(30*time.Second),
)
defer c.Close()

// Make requests
resp, err := c.Get(ctx, "/groups")
if err != nil {
    log.Fatal(err)
}
```

### Direct Client Construction with Bearer Token

```go
import (
    "github.com/jtimothystewart/dtiam/internal/client"
    "github.com/jtimothystewart/dtiam/internal/auth"
)

// Create static token manager (for bearer tokens)
// WARNING: Token will NOT auto-refresh!
tokenMgr := auth.NewStaticTokenManager("dt0c01.XXXX.YYYY...")

// Create client
c := client.New(
    "abc-123-def",
    tokenMgr,
)
defer c.Close()
```

### Custom Retry Configuration

```go
import "github.com/jtimothystewart/dtiam/internal/client"

retryConfig := client.RetryConfig{
    MaxRetries:      5,
    InitialDelay:    500 * time.Millisecond,
    MaxDelay:        30 * time.Second,
    ExponentialBase: 2.0,
    RetryStatuses:   []int{429, 500, 502, 503, 504},
}

c := client.New(
    "abc-123-def",
    tokenMgr,
    client.WithRetryConfig(retryConfig),
)
```

### Error Handling

```go
import "github.com/jtimothystewart/dtiam/internal/client"

resp, err := c.Get(ctx, "/groups/invalid-uuid")
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("Status: %d\n", apiErr.StatusCode)
        fmt.Printf("Message: %s\n", apiErr.Message)
        fmt.Printf("Body: %s\n", apiErr.ResponseBody)

        if apiErr.IsNotFound() {
            fmt.Println("Resource not found")
        }
    }
}
```

## Resource Handlers

### Groups

```go
import (
    "context"
    "github.com/jtimothystewart/dtiam/internal/resources"
)

handler := resources.NewGroupHandler(c)
ctx := context.Background()

// List all groups
groups, err := handler.List(ctx, nil)

// Get by UUID
group, err := handler.Get(ctx, "uuid-here")

// Get by name
group, err := handler.GetByName(ctx, "DevOps Team")

// Create a group
newGroup, err := handler.Create(ctx, "New Team", "A new team description")

// Update a group
updated, err := handler.Update(ctx, "uuid-here", "Updated Name", "Updated description")

// Delete a group
err = handler.Delete(ctx, "uuid-here")

// Member operations
members, err := handler.GetMembers(ctx, "group-uuid")
err = handler.AddMember(ctx, "group-uuid", "user@example.com")
err = handler.RemoveMember(ctx, "group-uuid", "user-uid")
```

### Users

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewUserHandler(c)
ctx := context.Background()

// List all users
users, err := handler.List(ctx, nil)

// Get by UID
user, err := handler.Get(ctx, "user-uid")

// Get by email
user, err := handler.GetByEmail(ctx, "user@example.com")

// Get user's groups
groups, err := handler.GetGroups(ctx, "user-uid")

// Create a user
firstName := "John"
lastName := "Doe"
newUser, err := handler.Create(ctx, "user@example.com", &firstName, &lastName, []string{"group-uuid"})

// Add to groups
err = handler.AddToGroups(ctx, "user@example.com", []string{"group-uuid-1", "group-uuid-2"})

// Remove from groups
err = handler.RemoveFromGroups(ctx, "user@example.com", []string{"group-uuid"})

// Replace all group memberships
err = handler.ReplaceGroups(ctx, "user@example.com", []string{"new-group-uuid"})

// Delete a user
err = handler.Delete(ctx, "user-uid")
```

### Service Users

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewServiceUserHandler(c)
ctx := context.Background()

// List all service users
serviceUsers, err := handler.List(ctx, nil)

// Get by UID
user, err := handler.Get(ctx, "service-user-uid")

// Get by name
user, err := handler.GetByName(ctx, "CI Pipeline")

// Create a service user (returns client credentials!)
description := "CI/CD automation"
result, err := handler.Create(ctx, "CI Pipeline", &description, []string{"group-uuid"})
// IMPORTANT: Save result["clientId"] and result["clientSecret"]
// The secret cannot be retrieved later!

// Update a service user
newName := "New Name"
updated, err := handler.Update(ctx, "service-user-uid", &newName, nil, nil)

// Delete a service user
err = handler.Delete(ctx, "service-user-uid")

// Group management
groups, err := handler.GetGroups(ctx, "service-user-uid")
err = handler.AddToGroup(ctx, "service-user-uid", "group-uuid")
err = handler.RemoveFromGroup(ctx, "service-user-uid", "group-uuid")
```

### Policies

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

// Account-level policies
handler := resources.NewPolicyHandler(c, "account", accountUUID)
ctx := context.Background()

// Global policies (read-only)
globalHandler := resources.NewPolicyHandler(c, "global", "global")

// List policies
policies, err := handler.List(ctx, nil)

// Get by UUID
policy, err := handler.Get(ctx, "policy-uuid")

// Get by name
policy, err := handler.GetByName(ctx, "admin-policy")

// Create a policy
newPolicy, err := handler.Create(ctx, map[string]any{
    "name":           "viewer-policy",
    "description":    "Read-only access",
    "statementQuery": "ALLOW settings:objects:read;",
})

// Delete a policy
err = handler.Delete(ctx, "policy-uuid")
```

### Bindings

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewBindingHandler(c)
ctx := context.Background()

// List all bindings (flattened)
bindings, err := handler.List(ctx, nil)

// Get bindings for a group
groupBindings, err := handler.GetForGroup(ctx, "group-uuid")

// Create a binding
err = handler.Create(ctx, "group-uuid", "policy-uuid", []string{"boundary-uuid"})

// Delete a binding
err = handler.Delete(ctx, "group-uuid", "policy-uuid")

// Add boundary to existing binding
err = handler.AddBoundary(ctx, "group-uuid", "policy-uuid", "boundary-uuid")

// Remove boundary from binding
err = handler.RemoveBoundary(ctx, "group-uuid", "policy-uuid", "boundary-uuid")
```

### Boundaries

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewBoundaryHandler(c)
ctx := context.Background()

// List boundaries
boundaries, err := handler.List(ctx, nil)

// Get by UUID
boundary, err := handler.Get(ctx, "boundary-uuid")

// Get by name
boundary, err := handler.GetByName(ctx, "prod-boundary")

// Create from management zones
desc := "Restricts to production zones"
boundary, err := handler.Create(ctx, "Production Only", []string{"Production", "Staging"}, nil, &desc)

// Create with custom query
query := "environment.tag.equals('production')"
desc2 := "Custom boundary query"
boundary, err := handler.Create(ctx, "Custom Boundary", nil, &query, &desc2)

// Get attached policies
attached, err := handler.GetAttachedPolicies(ctx, "boundary-uuid")

// Delete a boundary
err = handler.Delete(ctx, "boundary-uuid")
```

### Environments

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewEnvironmentHandler(c)
ctx := context.Background()

// List environments
environments, err := handler.List(ctx, nil)

// Get by ID
env, err := handler.Get(ctx, "env-id")

// Get by name
env, err := handler.GetByName(ctx, "Production")
```

### Account Limits

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewLimitsHandler(c)
ctx := context.Background()

// List all limits
limits, err := handler.List(ctx, nil)

// Get a specific limit
limit, err := handler.Get(ctx, "maxUsers")

// Check capacity before adding resources
hasCapacity, remaining, err := handler.CheckCapacity(ctx, "maxUsers", 10)
if hasCapacity {
    fmt.Printf("Capacity available (%d remaining)\n", remaining)
}
```

### Subscriptions

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewSubscriptionHandler(c)
ctx := context.Background()

// List all subscriptions
subscriptions, err := handler.List(ctx, nil)

// Get a specific subscription
sub, err := handler.Get(ctx, "subscription-uuid")

// Get usage forecast
forecast, err := handler.GetForecast(ctx)
```

### TokenHandler

Platform token operations. Requires `platform-token:tokens:manage` scope.

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewTokenHandler(c)
ctx := context.Background()

// List all platform tokens
tokens, err := handler.List(ctx, nil)

// Get a specific token by ID
token, err := handler.Get(ctx, "token-id")

// Create a new token (value only returned once!)
token, err := handler.Create(ctx, "CI Token", []string{"account-idm-read"}, "30d")

// Delete a token
err := handler.Delete(ctx, "token-id")
```

### AppHandler

App Engine Registry operations. Requires environment URL and `app-engine:apps:run` scope.

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

// env-id is auto-expanded to full URL
handler := resources.NewAppHandler(c, "abc12345")

// List all apps
apps, err := handler.List(ctx, nil)

// Get a specific app
app, err := handler.Get(ctx, "dynatrace.dashboards")

// Get all app IDs (useful for policy statements)
ids, err := handler.GetIDs(ctx)
```

### SchemaHandler

Settings 2.0 schema operations. Requires environment URL and `settings.read` scope.

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewSchemaHandler(c, "abc12345")

// List all schemas
schemas, err := handler.List(ctx, nil)

// Get a specific schema
schema, err := handler.Get(ctx, "builtin:alerting.profile")

// Search by pattern
matches, err := handler.Search(ctx, "alerting")

// Get only builtin schema IDs
builtinIDs, err := handler.GetBuiltinIDs(ctx)
```

## Output Formatting

### Using the Printer

```go
import "github.com/jtimothystewart/dtiam/internal/output"

// Create printer
printer := output.NewPrinter(output.FormatTable, false)

// Print data
data := []map[string]any{
    {"uuid": "123", "name": "Test"},
}
err := printer.Print(data, nil)

// With custom columns
columns := output.GroupColumns()
err = printer.Print(data, columns)
```

### Output Formats

```go
import "github.com/jtimothystewart/dtiam/internal/output"

// Available formats
output.FormatTable   // ASCII table (default)
output.FormatWide    // Table with extra columns
output.FormatJSON    // JSON output
output.FormatYAML    // YAML output
output.FormatCSV     // CSV output
output.FormatPlain   // Machine-readable plain text
```

### Custom Columns

```go
import "github.com/jtimothystewart/dtiam/internal/output"

// Basic column
col := output.Column{Key: "name", Header: "NAME"}

// Wide-only column (hidden in normal table view)
col := output.Column{Key: "createdAt", Header: "CREATED", WideOnly: true}

// Use predefined columns
groupCols := output.GroupColumns()
userCols := output.UserColumns()
policyCols := output.PolicyColumns()
```

## Authentication

dtiam supports two authentication methods:

### Option 1: OAuth2 Token Manager (Recommended)

The `OAuthTokenManager` handles OAuth2 client credentials flow with automatic token refresh. This is recommended for automation and long-running processes.

```go
import "github.com/jtimothystewart/dtiam/internal/auth"

// Create manager with OAuth2 credentials
tokenMgr := auth.NewOAuthTokenManager(
    "dt0s01.XXXX",           // client ID
    "dt0s01.XXXX.YYYY",      // client secret
    "abc-123-def",           // account UUID
)

// Get authentication token (auto-refreshes if expired)
token, err := tokenMgr.GetToken()

// Check if token is valid
isValid := tokenMgr.IsValid()
```

### Option 2: Static Bearer Token

The `StaticTokenManager` uses a pre-existing bearer token. **Warning:** Static tokens do NOT auto-refresh and will fail when expired.

```go
import "github.com/jtimothystewart/dtiam/internal/auth"

// Create manager with static bearer token
// WARNING: Token will NOT auto-refresh!
tokenMgr := auth.NewStaticTokenManager("dt0c01.XXXX.YYYY...")

// Get token (always returns the same token)
token, err := tokenMgr.GetToken()
```

**When to use Static Bearer Token:**
- Quick testing and debugging
- Interactive sessions with short-lived tokens
- Integration with external token providers
- One-off operations

**When NOT to use Static Bearer Token:**
- Automation scripts (use OAuth2)
- CI/CD pipelines (use OAuth2)
- Long-running processes (use OAuth2)
- Production environments (use OAuth2)

## Name Resolution

The `GetOrResolve` helper function handles resolution of identifiers that could be UUIDs or names:

```go
import "github.com/jtimothystewart/dtiam/internal/resources"

handler := resources.NewGroupHandler(c)
ctx := context.Background()

// Works with UUID or name
group, err := resources.GetOrResolve(ctx, handler, "DevOps Team")
// or
group, err := resources.GetOrResolve(ctx, handler, "550e8400-e29b-41d4-a716-446655440000")
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jtimothystewart/dtiam/internal/auth"
    "github.com/jtimothystewart/dtiam/internal/client"
    "github.com/jtimothystewart/dtiam/internal/output"
    "github.com/jtimothystewart/dtiam/internal/resources"
)

func main() {
    // Create token manager
    tokenMgr := auth.NewOAuthTokenManager(
        "dt0s01.XXXX",
        "dt0s01.XXXX.YYYY",
        "abc-123-def",
    )

    // Create client
    c := client.New("abc-123-def", tokenMgr)
    defer c.Close()

    ctx := context.Background()

    // Initialize handlers
    groupHandler := resources.NewGroupHandler(c)
    policyHandler := resources.NewPolicyHandler(c, "account", "abc-123-def")
    bindingHandler := resources.NewBindingHandler(c)

    // Create a group
    group, err := groupHandler.Create(ctx, "Platform Team", "Platform engineering team")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created group: %s\n", group["name"])

    // Find a policy
    policy, err := policyHandler.GetByName(ctx, "developer-policy")
    if err != nil {
        log.Fatal(err)
    }
    if policy == nil {
        log.Fatal("Policy not found!")
    }

    // Create binding
    groupUUID := group["uuid"].(string)
    policyUUID := policy["uuid"].(string)
    err = bindingHandler.Create(ctx, groupUUID, policyUUID, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Bound policy: %s\n", policy["name"])

    // List all groups with output formatting
    printer := output.NewPrinter(output.FormatTable, false)
    groups, _ := groupHandler.List(ctx, nil)
    printer.Print(groups, output.GroupColumns())
}
```

## See Also

- [Architecture](ARCHITECTURE.md)
- [Command Reference](COMMANDS.md)
- [Quick Start Guide](QUICK_START.md)
