package client

// API base URLs for Dynatrace services.
const (
	// IAMBaseURL is the root URL for the IAM API.
	IAMBaseURL = "https://api.dynatrace.com/iam/v1"

	// AccountsBasePath is the accounts path under the IAM API.
	AccountsBasePath = IAMBaseURL + "/accounts"

	// RepoBasePath is the repo path under the IAM API for level-scoped resources.
	RepoBasePath = IAMBaseURL + "/repo"

	// ResolutionBasePath is the resolution path for effective permissions.
	ResolutionBasePath = IAMBaseURL + "/resolution"

	// EnvBaseURL is the base URL for the Environment API.
	EnvBaseURL = "https://api.dynatrace.com/env/v2/accounts"

	// SubBaseURL is the base URL for the Subscription API.
	SubBaseURL = "https://api.dynatrace.com/sub/v2/accounts"

	// SSOTokenURL is the Dynatrace SSO token endpoint.
	SSOTokenURL = "https://sso.dynatrace.com/sso/oauth2/token"
)
