package cloner

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joshmedeski/sesh/v2/connector"
	"github.com/joshmedeski/sesh/v2/git"
	"github.com/joshmedeski/sesh/v2/github"
	"github.com/joshmedeski/sesh/v2/model"
)

type Cloner interface {
	// Clones a git repository
	Clone(opts model.GitCloneOptions) (string, error)
}

type RealCloner struct {
	connector connector.Connector
	git       git.Git
	config    model.Config
	shorthand github.ShorthandConverter
}

func NewCloner(connector connector.Connector, git git.Git, config model.Config) Cloner {
	return &RealCloner{
		connector: connector,
		git:       git,
		config:    config,
		shorthand: github.NewShorthandConverter(),
	}
}

func (c *RealCloner) Clone(opts model.GitCloneOptions) (string, error) {
	var repoURL string
	var clonePath string
	
	if c.shorthand.IsGitHubShorthand(opts.Repo) {
		// Handle GitHub shorthand
		var err error
		repoURL, err = c.shorthand.ConvertToURL(opts.Repo, c.config.GitHub)
		if err != nil {
			return "", err
		}
		
		// For GitHub repos, use smart clone path if not specified
		if opts.CmdDir == "" && opts.Dir == "" {
			org, repo, err := c.shorthand.ExtractOrgAndRepo(opts.Repo)
			if err != nil {
				return "", err
			}
			clonePath = c.shorthand.GetClonePath(org, repo, c.config.GitHub)
			
			// Split clonePath into cmdDir and dir
			lastSlash := strings.LastIndex(clonePath, "/")
			if lastSlash > 0 {
				opts.CmdDir = clonePath[:lastSlash]
				opts.Dir = clonePath[lastSlash+1:]
			}
		}
	} else {
		// Handle non-GitHub git repositories
		repoURL = opts.Repo
		
		// If no custom path specified, organize by domain
		if opts.CmdDir == "" && opts.Dir == "" {
			domain, org, repo, err := c.parseGitURL(opts.Repo)
			if err == nil && domain != "" && org != "" && repo != "" {
				cloneDir := c.config.GitHub.CloneDir
				if cloneDir == "" {
					cloneDir = "~/git"
				}
				cloneDir = c.expandHome(cloneDir)
				
				// Create domain-based path: ~/git/domain.com/org/repo
				clonePath = strings.Join([]string{cloneDir, domain, org, repo}, "/")
				lastSlash := strings.LastIndex(clonePath, "/")
				if lastSlash > 0 {
					opts.CmdDir = clonePath[:lastSlash]
					opts.Dir = clonePath[lastSlash+1:]
				}
			}
		}
	}

	// Create parent directory if it doesn't exist
	if opts.CmdDir != "" {
		if err := os.MkdirAll(opts.CmdDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", opts.CmdDir, err)
		}
	}

	if _, err := c.git.Clone(repoURL, opts.CmdDir, opts.Dir); err != nil {
		return "", err
	}

	path := getPath(opts)

	newOpts := model.ConnectOpts{}
	if _, err := c.connector.Connect(path, newOpts); err != nil {
		return "", err
	}

	return path, nil
}

func getPath(opts model.GitCloneOptions) string {
	var path string
	if opts.CmdDir != "" {
		path = opts.CmdDir
	} else {
		path, _ = os.Getwd()
	}

	if opts.Dir != "" {
		path = path + "/" + opts.Dir
	} else {
		repoName := getRepoName(opts.Repo)
		path = path + "/" + repoName
	}
	return path
}

func getRepoName(url string) string {
	parts := strings.Split(url, "/")
	lastPart := parts[len(parts)-1]
	repoName := strings.TrimSuffix(lastPart, ".git")
	return repoName
}

// parseGitURL extracts domain, org, and repo from a git URL
func (c *RealCloner) parseGitURL(gitURL string) (domain, org, repo string, err error) {
	// Handle SSH URLs like git@domain.com:org/repo.git
	if strings.HasPrefix(gitURL, "git@") {
		parts := strings.Split(gitURL, ":")
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid SSH git URL format")
		}
		
		domain = strings.Split(parts[0], "@")[1]
		pathParts := strings.Split(strings.TrimSuffix(parts[1], ".git"), "/")
		if len(pathParts) >= 2 {
			org = pathParts[0]
			repo = pathParts[1]
		}
		return domain, org, repo, nil
	}
	
	// Handle HTTPS URLs
	u, err := url.Parse(gitURL)
	if err != nil {
		return "", "", "", err
	}
	
	domain = u.Host
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) >= 2 {
		org = pathParts[0]
		repo = strings.TrimSuffix(pathParts[1], ".git")
	}
	
	return domain, org, repo, nil
}

// expandHome expands ~ to user home directory
func (c *RealCloner) expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return strings.Replace(path, "~", homeDir, 1)
		}
	}
	return path
}
