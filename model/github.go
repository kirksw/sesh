package model

import (
	"os"
	"time"
)

type GitHubRepo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	HTMLURL     string `json:"html_url"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	Archived    bool   `json:"archived"`
	Disabled    bool   `json:"disabled"`
	Language    string `json:"language"`
	UpdatedAt   string `json:"updated_at"`
	PushedAt    string `json:"pushed_at"`
	Topics      []string `json:"topics"`
}

type GitHubConfig struct {
	// Deprecated: Use Organizations instead
	Organization      string             `toml:"organization"`
	Organizations     []GitHubOrgConfig  `toml:"organizations"`
	Token             string             `toml:"token"`
	CacheTimeout      int                `toml:"cache_timeout"` // in minutes
	CloneDir          string             `toml:"clone_dir"`
	UseSSH            bool               `toml:"use_ssh"`
	IncludePersonal   bool               `toml:"include_personal"` // whether to include personal repos in addition to orgs
	ShowUncloned      *bool              `toml:"show_uncloned"`    // whether to show uncloned repos when using --github flag (default: true)
	ShowDescription   *bool              `toml:"show_description"` // whether to show repository descriptions (default: true)
}

type GitHubOrgConfig struct {
	Name        string `toml:"name"`
	DisplayName string `toml:"display_name"` // Optional: how to display this org in the list
	Token       string `toml:"token"`        // Optional: org-specific token
}

type GitHubCache struct {
	Repos     []GitHubRepo `json:"repos"`
	CachedAt  time.Time    `json:"cached_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// GetOrganizations returns all configured organizations, including legacy single org config
func (c GitHubConfig) GetOrganizations() []GitHubOrgConfig {
	var orgs []GitHubOrgConfig
	
	// Add organizations from new config format
	orgs = append(orgs, c.Organizations...)
	
	// Add legacy single organization if specified and not already in organizations list
	if c.Organization != "" {
		found := false
		for _, org := range orgs {
			if org.Name == c.Organization {
				found = true
				break
			}
		}
		if !found {
			legacyOrg := GitHubOrgConfig{
				Name:        c.Organization,
				DisplayName: c.Organization,
				Token:       c.Token, // Use global token for legacy org
			}
			orgs = append(orgs, legacyOrg)
		}
	}
	
	return orgs
}

// GetTokenForOrg returns the appropriate token for an organization
func (c GitHubConfig) GetTokenForOrg(orgName string) string {
	// Check for org-specific token first
	for _, org := range c.Organizations {
		if org.Name == orgName && org.Token != "" {
			return org.Token
		}
	}
	
	// Fall back to global token if available
	if c.Token != "" {
		return c.Token
	}
	
	// Fall back to GITHUB_TOKEN environment variable
	return os.Getenv("GITHUB_TOKEN")
}

// ShouldShowDescription returns whether to show repository descriptions
// Default is true to maintain backward compatibility
func (c GitHubConfig) ShouldShowDescription() bool {
	// If ShowDescription is nil (not configured), default to true
	// If ShowDescription is explicitly set, use that value
	if c.ShowDescription == nil {
		return true // Default behavior
	}
	return *c.ShowDescription
}

// ShouldShowUncloned returns whether to show uncloned repositories in --github mode
// Default is true to make the --github flag useful for discovery
func (c GitHubConfig) ShouldShowUncloned() bool {
	// If ShowUncloned is nil (not configured), default to true
	// If ShowUncloned is explicitly set, use that value
	if c.ShowUncloned == nil {
		return true // Default behavior - show all repos for discovery
	}
	return *c.ShowUncloned
}


