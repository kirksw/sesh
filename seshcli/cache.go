package seshcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joshmedeski/sesh/v2/github"
)

func NewCacheCommand(githubCache github.Cache) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage sesh cache",
	}

	// Add subcommands
	cmd.AddCommand(
		NewCacheClearCommand(githubCache),
		NewCacheInfoCommand(githubCache),
	)

	return cmd
}

func NewCacheClearCommand(githubCache github.Cache) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clear",
		Aliases: []string{"clean"},
		Short:   "Clear GitHub cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			github, _ := cmd.Flags().GetBool("github")
			
			if github || len(args) == 0 {
				if err := clearGitHubCache(githubCache); err != nil {
					return fmt.Errorf("failed to clear GitHub cache: %w", err)
				}
				fmt.Println("✅ GitHub cache cleared")
			}
			
			return nil
		},
	}

	cmd.Flags().BoolP("github", "g", true, "clear GitHub cache")

	return cmd
}

func NewCacheInfoCommand(githubCache github.Cache) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show cache information",
		RunE: func(cmd *cobra.Command, args []string) error {
			cachePath := githubCache.GetCachePath()
			fmt.Printf("Cache directory: %s\n", cachePath)
			
			if _, err := os.Stat(cachePath); os.IsNotExist(err) {
				fmt.Println("Cache directory does not exist")
				return nil
			}
			
			files, err := os.ReadDir(cachePath)
			if err != nil {
				return fmt.Errorf("failed to read cache directory: %w", err)
			}
			
			if len(files) == 0 {
				fmt.Println("No cache files found")
				return nil
			}
			
			fmt.Printf("Cache files (%d):\n", len(files))
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
					info, _ := file.Info()
					orgName := file.Name()[:len(file.Name())-5] // Remove .json extension
					fmt.Printf("  %s (modified: %s)\n", orgName, info.ModTime().Format("2006-01-02 15:04:05"))
				}
			}
			
			return nil
		},
	}

	return cmd
}

func clearGitHubCache(githubCache github.Cache) error {
	cachePath := githubCache.GetCachePath()
	
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil // Cache directory doesn't exist, nothing to clear
	}
	
	files, err := os.ReadDir(cachePath)
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(cachePath, file.Name())
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to remove cache file %s: %w", file.Name(), err)
			}
		}
	}
	
	return nil
}
