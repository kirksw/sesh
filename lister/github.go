package lister

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshmedeski/sesh/v2/github"
	"github.com/joshmedeski/sesh/v2/model"
)

type GitHub interface {
	ListRepos(org string) ([]model.GitHubRepo, error)
	ListAllRepos(config model.GitHubConfig) (map[string][]model.GitHubRepo, error)
	ListAllReposWithRefresh(config model.GitHubConfig, refresh bool) (map[string][]model.GitHubRepo, error)
	GetAuthenticatedUsername(token string) (string, error)
}

type RealGitHub struct {
	client github.Client
	cache  github.Cache
}

func NewGitHub(client github.Client, cache github.Cache) GitHub {
	return &RealGitHub{
		client: client,
		cache:  cache,
	}
}

func (g *RealGitHub) ListRepos(org string) ([]model.GitHubRepo, error) {
	// Try cache first
	if repos, found := g.cache.Get(org); found {
		return repos, nil
	}

	slog.Debug("Cache miss, fetching from GitHub API", "org", org)
	
	// Fetch from GitHub API
	repos, err := g.client.ListOrgRepos(org)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos from GitHub: %w", err)
	}

	// Cache the results (default to 30 minutes if not configured)
	g.cache.Set(org, repos, 30)

	return repos, nil
}

func (g *RealGitHub) ListAllRepos(config model.GitHubConfig) (map[string][]model.GitHubRepo, error) {
	return g.ListAllReposWithRefresh(config, false)
}

func (g *RealGitHub) ListAllReposWithRefresh(config model.GitHubConfig, refresh bool) (map[string][]model.GitHubRepo, error) {
	orgs := config.GetOrganizations()
	results := make(map[string][]model.GitHubRepo)
	
	for _, orgConfig := range orgs {
		var repos []model.GitHubRepo
		var err error
		
		// Try cache first unless refresh is requested
		if !refresh {
			if cachedRepos, found := g.cache.Get(orgConfig.Name); found {
				results[orgConfig.Name] = cachedRepos
				continue
			}
			slog.Debug("Cache miss, fetching from GitHub API", "org", orgConfig.Name)
		} else {
			slog.Debug("Cache refresh requested, fetching from GitHub API", "org", orgConfig.Name)
		}
		
		// Get the appropriate token for this org
		token := config.GetTokenForOrg(orgConfig.Name)
		
		// Try organization endpoint first, fall back to user endpoint if 404
		repos, err = g.client.ListOrgReposWithToken(orgConfig.Name, token)
		if err != nil {
			// Check if it's a 404 error (not an organization)
			if strings.Contains(err.Error(), "404") {
				slog.Debug("Organization not found, trying user endpoint", "org", orgConfig.Name)
				// Try as user instead
				repos, err = g.client.ListUserReposWithToken(orgConfig.Name, token)
				if err != nil {
					slog.Error("Failed to fetch repos from GitHub (both org and user endpoints)", "org", orgConfig.Name, "error", err)
					continue // Continue with other orgs instead of failing completely
				}
			} else {
				slog.Error("Failed to fetch repos from GitHub", "org", orgConfig.Name, "error", err)
				continue // Continue with other orgs instead of failing completely
			}
		}

		// Cache the results
		cacheTimeout := config.CacheTimeout
		if cacheTimeout == 0 {
			cacheTimeout = 30 // Default to 30 minutes
		}
		g.cache.Set(orgConfig.Name, repos, cacheTimeout)
		
		results[orgConfig.Name] = repos
	}

	// Include personal repos if enabled and we have a token
	if config.IncludePersonal {
		token := config.GetTokenForOrg("") // Get the global token or GITHUB_TOKEN
		if token != "" {
			// Get the authenticated user's username
			username, err := g.GetAuthenticatedUsername(token)
			if err != nil {
				slog.Error("Failed to get authenticated username for personal repos", "error", err)
			} else {
				// Try cache first unless refresh is requested
				if !refresh {
					if cachedRepos, found := g.cache.Get(username); found {
						results[username] = cachedRepos
					} else {
						slog.Debug("Cache miss, fetching personal repos from GitHub API", "username", username)
						personalRepos, err := g.fetchPersonalRepos(token)
						if err != nil {
							slog.Error("Failed to fetch personal repos from GitHub", "error", err)
						} else {
							// Cache the results
							cacheTimeout := config.CacheTimeout
							if cacheTimeout == 0 {
								cacheTimeout = 30 // Default to 30 minutes
							}
							g.cache.Set(username, personalRepos, cacheTimeout)
							results[username] = personalRepos
						}
					}
				} else {
					slog.Debug("Cache refresh requested, fetching personal repos from GitHub API", "username", username)
					personalRepos, err := g.fetchPersonalRepos(token)
					if err != nil {
						slog.Error("Failed to fetch personal repos from GitHub", "error", err)
					} else {
						// Cache the results
						cacheTimeout := config.CacheTimeout
						if cacheTimeout == 0 {
							cacheTimeout = 30 // Default to 30 minutes
						}
						g.cache.Set(username, personalRepos, cacheTimeout)
						results[username] = personalRepos
					}
				}
			}
		}
	}

	return results, nil
}

// GetAuthenticatedUsername gets the username of the authenticated user
func (g *RealGitHub) GetAuthenticatedUsername(token string) (string, error) {
	return g.client.GetAuthenticatedUsername(token)
}

// fetchPersonalRepos fetches repositories for the authenticated user
func (g *RealGitHub) fetchPersonalRepos(token string) ([]model.GitHubRepo, error) {
	return g.client.ListAuthenticatedUserReposWithToken(token)
}

func listGitHub(l *RealLister, opts ListOptions) (model.SeshSessions, error) {
	config := l.config.GitHub
	orgs := config.GetOrganizations()
	
	if len(orgs) == 0 {
		slog.Debug("No GitHub organizations configured, skipping GitHub repos")
		return model.SeshSessions{
			Directory:    make(model.SeshSessionMap),
			OrderedIndex: []string{},
		}, nil
	}

	allRepos, err := l.github.ListAllReposWithRefresh(config, opts.Refresh)
	if err != nil {
		return model.SeshSessions{}, fmt.Errorf("couldn't list GitHub repos: %w", err)
	}

	orderedIndex := make([]string, 0)
	directory := make(model.SeshSessionMap)

	// Process repos from each organization
	for _, orgConfig := range orgs {
		repos, exists := allRepos[orgConfig.Name]
		if !exists {
			continue
		}

		for _, repo := range repos {
			// Skip archived, disabled, or fork repos unless configured otherwise
			if repo.Archived || repo.Disabled {
				continue
			}

			// Generate session name with org prefix for disambiguation
			displayName := orgConfig.DisplayName
			if displayName == "" {
				displayName = orgConfig.Name
			}
			
			name := fmt.Sprintf("%s/%s", displayName, repo.Name)
			if repo.Description != "" && config.ShouldShowDescription() {
				name = fmt.Sprintf("%s/%s (%s)", displayName, repo.Name, repo.Description)
			}

			// Determine clone path
			cloneDir := config.CloneDir
			if cloneDir == "" {
				homeDir, _ := os.UserHomeDir()
				cloneDir = filepath.Join(homeDir, "git")
			} else if strings.HasPrefix(cloneDir, "~/") {
				// Expand tilde to home directory
				homeDir, _ := os.UserHomeDir()
				cloneDir = filepath.Join(homeDir, cloneDir[2:])
			}
			clonePath := filepath.Join(cloneDir, "github.com", orgConfig.Name, repo.Name)

			// Check if repo is already cloned
			var path string
			var exists bool
			if _, err := os.Stat(clonePath); err == nil {
				path = clonePath
				exists = true
			} else {
				// Use clone URL as path for uncloned repos
				if config.UseSSH {
					path = repo.SSHURL
				} else {
					path = repo.CloneURL
				}
			}

			// When using --github flag, show repos based on config (defaults to showing all repos)
			if !exists && !config.ShouldShowUncloned() {
				continue
			}

			key := fmt.Sprintf("github:%s/%s", orgConfig.Name, repo.Name)
			orderedIndex = append(orderedIndex, key)
			
			session := model.SeshSession{
				Src:  "github",
				Name: name,
				Path: path,
			}

			// Add metadata for GitHub repos
			if !exists {
				// For uncloned repos, we'll use a special startup command to clone first
				cloneCmd := fmt.Sprintf("git clone %s %s && cd %s", path, clonePath, clonePath)
				session.StartupCommand = cloneCmd
				session.Path = clonePath // Update path to where it will be cloned
			}

			directory[key] = session
		}
	}

	// Process personal repos if include_personal is enabled
	if config.IncludePersonal {
		token := config.GetTokenForOrg("") // Get the global token or GITHUB_TOKEN
		if token != "" {
			username, err := l.github.GetAuthenticatedUsername(token)
			if err == nil {
				if personalRepos, exists := allRepos[username]; exists {
					for _, repo := range personalRepos {
						// Skip archived, disabled, or fork repos unless configured otherwise
						if repo.Archived || repo.Disabled {
							continue
						}

						// Generate session name with username prefix
						name := fmt.Sprintf("%s/%s", username, repo.Name)
						if repo.Description != "" && config.ShouldShowDescription() {
							name = fmt.Sprintf("%s/%s (%s)", username, repo.Name, repo.Description)
						}

						// Determine clone path
						cloneDir := config.CloneDir
						if cloneDir == "" {
							homeDir, _ := os.UserHomeDir()
							cloneDir = filepath.Join(homeDir, "git")
						} else if strings.HasPrefix(cloneDir, "~/") {
							// Expand tilde to home directory
							homeDir, _ := os.UserHomeDir()
							cloneDir = filepath.Join(homeDir, cloneDir[2:])
						}
						clonePath := filepath.Join(cloneDir, "github.com", username, repo.Name)

						// Check if repo is already cloned
						var path string
						var exists bool
						if _, err := os.Stat(clonePath); err == nil {
							path = clonePath
							exists = true
						} else {
							// Use clone URL as path for uncloned repos
							if config.UseSSH {
								path = repo.SSHURL
							} else {
								path = repo.CloneURL
							}
						}

						// When using --github flag, show repos based on config (defaults to showing all repos)
						if !exists && !config.ShouldShowUncloned() {
							continue
						}

						key := fmt.Sprintf("github:%s/%s", username, repo.Name)
						orderedIndex = append(orderedIndex, key)
						
						session := model.SeshSession{
							Src:  "github",
							Name: name,
							Path: path,
						}

						// Add metadata for GitHub repos
						if !exists {
							// For uncloned repos, we'll use a special startup command to clone first
							cloneCmd := fmt.Sprintf("git clone %s %s && cd %s", path, clonePath, clonePath)
							session.StartupCommand = cloneCmd
							session.Path = clonePath // Update path to where it will be cloned
						}

						directory[key] = session
					}
				}
			}
		}
	}

	return model.SeshSessions{
		Directory:    directory,
		OrderedIndex: orderedIndex,
	}, nil
}

func (l *RealLister) FindGitHubSession(name string) (model.SeshSession, bool) {
	// List GitHub sessions including all repos since this is used for connecting
	sessions, err := listGitHub(l, ListOptions{GitHub: true})
	if err != nil {
		return model.SeshSession{}, false
	}
	
	// Try to find by exact name match first
	for _, session := range sessions.Directory {
		if session.Name == name {
			return session, true
		}
	}
	
	// If not found by name, try to find by key
	if session, exists := sessions.Directory[name]; exists {
		return session, true
	}
	
	return model.SeshSession{}, false
}
