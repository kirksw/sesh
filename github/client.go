package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v66/github"
	"github.com/joshmedeski/sesh/v2/model"
	"golang.org/x/oauth2"
)

// Client interface for GitHub operations
type Client interface {
	ListOrgRepos(org string) ([]model.GitHubRepo, error)
	ListOrgReposWithToken(org, token string) ([]model.GitHubRepo, error)
	ListUserRepos(username string) ([]model.GitHubRepo, error)
	ListUserReposWithToken(username, token string) ([]model.GitHubRepo, error)
	ListAuthenticatedUserReposWithToken(token string) ([]model.GitHubRepo, error)
	GetAuthenticatedUsername(token string) (string, error)
}

// RealClient wraps the go-github client
type RealClient struct {
	defaultToken string
}

// NewClient creates a new GitHub client
func NewClient(token string) Client {
	return &RealClient{
		defaultToken: token,
	}
}

// createGitHubClient creates a go-github client with the given token
func (c *RealClient) createGitHubClient(token string) *github.Client {
	if token == "" {
		// Try to get token from environment if not provided
		token = os.Getenv("GITHUB_TOKEN")
	}
	
	if token == "" {
		// Return unauthenticated client (rate limited)
		return github.NewClient(nil)
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	
	return github.NewClient(tc)
}

// convertRepo converts a go-github repository to our model
func convertRepo(repo *github.Repository) model.GitHubRepo {
	var description, language string
	var topics []string
	
	if repo.Description != nil {
		description = *repo.Description
	}
	if repo.Language != nil {
		language = *repo.Language
	}
	if repo.Topics != nil {
		topics = repo.Topics
	}

	return model.GitHubRepo{
		ID:          int(*repo.ID),
		Name:        *repo.Name,
		FullName:    *repo.FullName,
		Description: description,
		CloneURL:    *repo.CloneURL,
		SSHURL:      *repo.SSHURL,
		HTMLURL:     *repo.HTMLURL,
		Private:     *repo.Private,
		Fork:        *repo.Fork,
		Archived:    *repo.Archived,
		Disabled:    *repo.Disabled,
		Language:    language,
		UpdatedAt:   repo.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		PushedAt:    repo.PushedAt.Format("2006-01-02T15:04:05Z"),
		Topics:      topics,
	}
}

// ListOrgRepos lists repositories for an organization using the default token
func (c *RealClient) ListOrgRepos(org string) ([]model.GitHubRepo, error) {
	return c.ListOrgReposWithToken(org, c.defaultToken)
}

// ListOrgReposWithToken lists repositories for an organization with a specific token
func (c *RealClient) ListOrgReposWithToken(org, token string) ([]model.GitHubRepo, error) {
	client := c.createGitHubClient(token)
	ctx := context.Background()
	
	var allRepos []model.GitHubRepo
	
	opts := &github.RepositoryListByOrgOptions{
		Type: "all", // public, private, forks, sources, member
		Sort: "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories for org %s: %w", org, err)
		}

		for _, repo := range repos {
			allRepos = append(allRepos, convertRepo(repo))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// ListUserRepos lists repositories for a user using the default token
func (c *RealClient) ListUserRepos(username string) ([]model.GitHubRepo, error) {
	return c.ListUserReposWithToken(username, c.defaultToken)
}

// ListAuthenticatedUserReposWithToken lists repositories for the authenticated user
func (c *RealClient) ListAuthenticatedUserReposWithToken(token string) ([]model.GitHubRepo, error) {
	client := c.createGitHubClient(token)
	ctx := context.Background()
	
	var allRepos []model.GitHubRepo
	
	opts := &github.RepositoryListOptions{
		Affiliation: "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := client.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories for authenticated user: %w", err)
		}

		for _, repo := range repos {
			allRepos = append(allRepos, convertRepo(repo))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// GetAuthenticatedUsername returns the username of the authenticated user
func (c *RealClient) GetAuthenticatedUsername(token string) (string, error) {
	client := c.createGitHubClient(token)
	ctx := context.Background()
	
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}
	
	if user.Login == nil {
		return "", fmt.Errorf("authenticated user login is nil")
	}
	
	return *user.Login, nil
}

// ListUserReposWithToken lists repositories for a user with a specific token
func (c *RealClient) ListUserReposWithToken(username, token string) ([]model.GitHubRepo, error) {
	client := c.createGitHubClient(token)
	ctx := context.Background()
	
	var allRepos []model.GitHubRepo
	
	// First, try to get the authenticated user to see if this is their own profile
	var isAuthenticatedUser bool
	if token != "" {
		if user, _, err := client.Users.Get(ctx, ""); err == nil && user.Login != nil && *user.Login == username {
			isAuthenticatedUser = true
		}
	}

	if isAuthenticatedUser {
		// Use authenticated user endpoint to get private repos
		opts := &github.RepositoryListOptions{
			Affiliation: "owner",
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		for {
			repos, resp, err := client.Repositories.List(ctx, "", opts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories for authenticated user %s: %w", username, err)
			}

			for _, repo := range repos {
				allRepos = append(allRepos, convertRepo(repo))
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	} else {
		// Use public user endpoint (only public repos)
		opts := &github.RepositoryListByUserOptions{
			Type: "all",
			Sort: "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		for {
			repos, resp, err := client.Repositories.ListByUser(ctx, username, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories for user %s: %w", username, err)
			}

			for _, repo := range repos {
				allRepos = append(allRepos, convertRepo(repo))
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	return allRepos, nil
}
