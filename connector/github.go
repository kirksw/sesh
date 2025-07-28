package connector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

func githubStrategy(c *RealConnector, name string) (model.Connection, error) {
	session, exists := c.lister.FindGitHubSession(name)
	if !exists {
		return model.Connection{Found: false}, nil
	}
	
	// Check if this is an uncloned repository that needs cloning
	if session.StartupCommand != "" && strings.Contains(session.StartupCommand, "git clone") {
		// Extract clone information from the startup command
		// The startup command format is: "git clone <url> <path> && cd <path>"
		parts := strings.Split(session.StartupCommand, " ")
		
		if len(parts) >= 4 && parts[0] == "git" && parts[1] == "clone" {
			repoURL := parts[2]
			clonePath := parts[3]
			
			// Create parent directory if it doesn't exist
			parentDir := filepath.Dir(clonePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return model.Connection{}, fmt.Errorf("failed to create parent directory: %w", err)
			}
			
			// Check if already cloned
			if _, err := os.Stat(clonePath); os.IsNotExist(err) {
				// Clone the repository using git directly
				if _, err := c.git.Clone(repoURL, parentDir, filepath.Base(clonePath)); err != nil {
					return model.Connection{}, fmt.Errorf("failed to clone repository: %w", err)
				}
			}
			
			// Update session to point to the cloned directory and remove startup command
			session.Path = clonePath
			session.StartupCommand = ""
		}
	}
	
	return model.Connection{
		Found:       true,
		Session:     session,
		New:         true, // GitHub sessions are always "new" since they create tmux sessions
		AddToZoxide: true,
	}, nil
}
