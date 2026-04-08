// Package config provides configuration management for the dtiam CLI.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config represents the root configuration structure.
type Config struct {
	APIVersion     string            `yaml:"api-version" json:"api-version"`
	Kind           string            `yaml:"kind" json:"kind"`
	CurrentContext string            `yaml:"current-context" json:"current-context"`
	Contexts       []NamedContext    `yaml:"contexts" json:"contexts"`
	Credentials    []NamedCredential `yaml:"credentials" json:"credentials"`
	Preferences    Preferences       `yaml:"preferences,omitempty" json:"preferences,omitempty"`
}

// NamedContext wraps a context with its name.
type NamedContext struct {
	Name    string  `yaml:"name" json:"name"`
	Context Context `yaml:"context" json:"context"`
}

// Context holds context configuration.
type Context struct {
	AccountUUID    string `yaml:"account-uuid" json:"account-uuid"`
	CredentialsRef string `yaml:"credentials-ref" json:"credentials-ref"`
}

// NamedCredential wraps a credential with its name.
type NamedCredential struct {
	Name       string     `yaml:"name" json:"name"`
	Credential Credential `yaml:"credential" json:"credential"`
}

// Credential holds OAuth2 credentials.
type Credential struct {
	ClientID         string `yaml:"client-id" json:"client-id"`
	ClientSecret     string `yaml:"client-secret" json:"client-secret"`
	APIURL           string `yaml:"api-url,omitempty" json:"api-url,omitempty"`
	Scopes           string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	EnvironmentURL   string `yaml:"environment-url,omitempty" json:"environment-url,omitempty"`
	EnvironmentToken string `yaml:"environment-token,omitempty" json:"environment-token,omitempty"`
}

// Preferences holds user preferences.
type Preferences struct {
	Output string `yaml:"output,omitempty" json:"output,omitempty"`
	Editor string `yaml:"editor,omitempty" json:"editor,omitempty"`
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		APIVersion:  "v1",
		Kind:        "Config",
		Contexts:    []NamedContext{},
		Credentials: []NamedCredential{},
		Preferences: Preferences{
			Output: "table",
			Editor: "vim",
		},
	}
}

// GetContext returns the context with the given name, or nil if not found.
func (c *Config) GetContext(name string) *Context {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i].Context
		}
	}
	return nil
}

// GetCurrentContext returns the current context, or nil if not set.
func (c *Config) GetCurrentContext() *Context {
	if c.CurrentContext == "" {
		return nil
	}
	return c.GetContext(c.CurrentContext)
}

// GetCredential returns the credential with the given name, or nil if not found.
func (c *Config) GetCredential(name string) *Credential {
	for i := range c.Credentials {
		if c.Credentials[i].Name == name {
			return &c.Credentials[i].Credential
		}
	}
	return nil
}

// GetCurrentCredential returns the credential for the current context, or nil if not set.
func (c *Config) GetCurrentCredential() *Credential {
	ctx := c.GetCurrentContext()
	if ctx == nil || ctx.CredentialsRef == "" {
		return nil
	}
	return c.GetCredential(ctx.CredentialsRef)
}

// SetContext sets or updates a context.
func (c *Config) SetContext(name string, accountUUID, credentialsRef *string) error {
	// Find existing context
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			if accountUUID != nil {
				c.Contexts[i].Context.AccountUUID = *accountUUID
			}
			if credentialsRef != nil {
				c.Contexts[i].Context.CredentialsRef = *credentialsRef
			}
			return nil
		}
	}

	// Create new context
	if accountUUID == nil {
		return fmt.Errorf("account-uuid is required for new context")
	}

	newCtx := NamedContext{
		Name: name,
		Context: Context{
			AccountUUID: *accountUUID,
		},
	}
	if credentialsRef != nil {
		newCtx.Context.CredentialsRef = *credentialsRef
	}

	c.Contexts = append(c.Contexts, newCtx)
	return nil
}

// SetCredential sets or updates a credential.
func (c *Config) SetCredential(name, clientID, clientSecret string) {
	// Find existing credential
	for i := range c.Credentials {
		if c.Credentials[i].Name == name {
			c.Credentials[i].Credential.ClientID = clientID
			c.Credentials[i].Credential.ClientSecret = clientSecret
			return
		}
	}

	// Create new credential
	c.Credentials = append(c.Credentials, NamedCredential{
		Name: name,
		Credential: Credential{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	})
}

// SetCredentialField updates a single field on an existing credential.
// Returns false if the credential does not exist.
func (c *Config) SetCredentialField(name, field, value string) bool {
	for i := range c.Credentials {
		if c.Credentials[i].Name == name {
			cred := &c.Credentials[i].Credential
			switch field {
			case "api-url":
				cred.APIURL = value
			case "scopes":
				cred.Scopes = value
			case "environment-url":
				cred.EnvironmentURL = value
			case "environment-token":
				cred.EnvironmentToken = value
			default:
				return false
			}
			return true
		}
	}
	return false
}

// GetEffectiveAPIURL returns the API base URL, checking env > credential > default.
func GetEffectiveAPIURL(cred *Credential, defaultURL string) string {
	if envURL := os.Getenv(EnvAPIURL); envURL != "" {
		return envURL
	}
	if cred != nil && cred.APIURL != "" {
		return cred.APIURL
	}
	return defaultURL
}

// GetEffectiveEnvironmentURL returns the environment URL, checking env > credential.
func GetEffectiveEnvironmentURL(cred *Credential) string {
	if envURL := os.Getenv(EnvEnvironmentURL); envURL != "" {
		return envURL
	}
	if cred != nil && cred.EnvironmentURL != "" {
		return cred.EnvironmentURL
	}
	return ""
}

// DeleteContext removes a context by name. Returns true if deleted.
func (c *Config) DeleteContext(name string) bool {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			if c.CurrentContext == name {
				c.CurrentContext = ""
			}
			return true
		}
	}
	return false
}

// DeleteCredential removes a credential by name. Returns true if deleted.
func (c *Config) DeleteCredential(name string) bool {
	for i := range c.Credentials {
		if c.Credentials[i].Name == name {
			c.Credentials = append(c.Credentials[:i], c.Credentials[i+1:]...)
			return true
		}
	}
	return false
}

// UseContext sets the current context. Returns error if context doesn't exist.
func (c *Config) UseContext(name string) error {
	if c.GetContext(name) == nil {
		return fmt.Errorf("context %q not found", name)
	}
	c.CurrentContext = name
	return nil
}

// GetContextNames returns all context names.
func (c *Config) GetContextNames() []string {
	names := make([]string, len(c.Contexts))
	for i := range c.Contexts {
		names[i] = c.Contexts[i].Name
	}
	return names
}

// GetCredentialNames returns all credential names.
func (c *Config) GetCredentialNames() []string {
	names := make([]string, len(c.Credentials))
	for i := range c.Credentials {
		names[i] = c.Credentials[i].Name
	}
	return names
}

// MaskSecret masks a secret string for display.
func MaskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.CurrentContext != "" && c.GetContext(c.CurrentContext) == nil {
		return fmt.Errorf("current context %q not found in contexts", c.CurrentContext)
	}

	// Check that credential refs exist
	for _, ctx := range c.Contexts {
		if ctx.Context.CredentialsRef != "" {
			if c.GetCredential(ctx.Context.CredentialsRef) == nil {
				return fmt.Errorf("context %q references credential %q which does not exist",
					ctx.Name, ctx.Context.CredentialsRef)
			}
		}
	}

	return nil
}
