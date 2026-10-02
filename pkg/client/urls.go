package client

// API base URLs for Dynatrace services.
const (
	// DefaultAPIHost is the host every Account Management API is served from.
	// Config.APIHost replaces it, so all account APIs move together when an
	// alternative host (for example a sprint or dev stage) is configured.
	DefaultAPIHost = "https://api.dynatrace.com"

	// IAMBaseURL is the root URL for the IAM API.
	IAMBaseURL = DefaultAPIHost + "/iam/v1"

	// AccountsBasePath is the accounts path under the IAM API.
	AccountsBasePath = IAMBaseURL + "/accounts"

	// RepoBasePath is the repo path under the IAM API for level-scoped resources.
	RepoBasePath = IAMBaseURL + "/repo"

	// ResolutionBasePath is the resolution path for effective permissions.
	ResolutionBasePath = IAMBaseURL + "/resolution"

	// EnvBaseURL is the base URL for the Environment API.
	EnvBaseURL = "https://api.dynatrace.com/env/v2/accounts"

	// SubBaseURL is the base URL for the Subscription API (v2).
	SubBaseURL = "https://api.dynatrace.com/sub/v2/accounts"

	// SubV3BaseURL is the base URL for Subscription API endpoints that moved to
	// v3. Only the cost-per-environment endpoint lives here; subscription
	// listing, usage and forecast remain on v2.
	SubV3BaseURL = "https://api.dynatrace.com/sub/v3/accounts"

	// AuditBaseURL is the base URL for the account audit log API.
	AuditBaseURL = "https://api.dynatrace.com/audit/v1/accounts"

	// RefBaseURL is the base URL for the reference data API. Unlike the other
	// account APIs this one is not scoped by account UUID.
	RefBaseURL = "https://api.dynatrace.com/ref/v1/account"

	// NotificationsBaseURL is the base URL for the account notifications API.
	// Note the unusual unprefixed /v1 path; this is not under /iam or /sub.
	NotificationsBaseURL = "https://api.dynatrace.com/v1/accounts"

	// SSOTokenURL is the Dynatrace SSO token endpoint.
	SSOTokenURL = "https://sso.dynatrace.com/sso/oauth2/token"
)
