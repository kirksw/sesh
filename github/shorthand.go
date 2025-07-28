package github

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

// GitHub shorthand patterns: org/repo, github.com/org/repo, etc.
var githubShorthandRegex = regexp.MustCompile(`^([a-zA-Z0-9._-]+)\/([a-zA-Z0-9._-]+)$`)
var githubURLRegex = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/([a-zA-Z0-9._-]+)/([a-zA-Z0-9._-]+)(?:\.git)?/?$`)

type ShorthandConverter interface {
	IsGitHubShorthand(input string) bool
	ConvertToURL(input string, config model.GitHubConfig) (string, error)
	ExtractOrgAndRepo(input string) (org, repo string, err error)
	GetClonePath(org, repo string, config model.GitHubConfig) string
}

type RealShorthandConverter struct{}

func NewShorthandConverter() ShorthandConverter {
	return &RealShorthandConverter{}
}

// IsGitHubShorthand checks if the input matches GitHub shorthand patterns
func (c *RealShorthandConverter) IsGitHubShorthand(input string) bool {
	// Check for org/repo pattern
	if githubShorthandRegex.MatchString(input) {
		return true
	}
	
	// Check for github.com URLs
	if githubURLRegex.MatchString(input) {
		return true
	}
	
	return false
}

// ConvertToURL converts GitHub shorthand to full clone URL
func (c *RealShorthandConverter) ConvertToURL(input string, config model.GitHubConfig) (string, error) {
	org, repo, err := c.ExtractOrgAndRepo(input)
	if err != nil {
		return "", err
	}
	
	if config.UseSSH {
		return fmt.Sprintf("git@github.com:%s/%s.git", org, repo), nil
	}
	
	return fmt.Sprintf("https://github.com/%s/%s.git", org, repo), nil
}

// ExtractOrgAndRepo extracts organization and repository names from various GitHub input formats
func (c *RealShorthandConverter) ExtractOrgAndRepo(input string) (string, string, error) {
	// Try shorthand pattern first (org/repo)
	if matches := githubShorthandRegex.FindStringSubmatch(input); len(matches) == 3 {
		return matches[1], matches[2], nil
	}
	
	// Try GitHub URL pattern
	if matches := githubURLRegex.FindStringSubmatch(input); len(matches) == 3 {
		return matches[1], matches[2], nil
	}
	
	return "", "", fmt.Errorf("invalid GitHub shorthand format: %s", input)
}

// GetClonePath determines where to clone the repository
func (c *RealShorthandConverter) GetClonePath(org, repo string, config model.GitHubConfig) string {
	if config.CloneDir != "" {
		return expandHome(fmt.Sprintf("%s/github.com/%s/%s", config.CloneDir, org, repo))
	}
	
	// Default to ~/git/github.com/org/repo
	return expandHome(fmt.Sprintf("~/git/github.com/%s/%s", org, repo))
}

// expandHome expands ~ to user home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return strings.Replace(path, "~", homeDir, 1)
		}
	}
	return path
}
