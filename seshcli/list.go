package seshcli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joshmedeski/sesh/v2/icon"
	"github.com/joshmedeski/sesh/v2/json"
	"github.com/joshmedeski/sesh/v2/lister"
	"github.com/joshmedeski/sesh/v2/model"
)

func NewListCommand(icon icon.Icon, json json.Json, list lister.Lister) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, _ := cmd.Flags().GetBool("config")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			tmux, _ := cmd.Flags().GetBool("tmux")
			zoxide, _ := cmd.Flags().GetBool("zoxide")
			hideAttached, _ := cmd.Flags().GetBool("hide-attached")
			icons, _ := cmd.Flags().GetBool("icons")
			tmuxinator, _ := cmd.Flags().GetBool("tmuxinator")
			github, _ := cmd.Flags().GetBool("github")
			hideDuplicates, _ := cmd.Flags().GetBool("hide-duplicates")
			refresh, _ := cmd.Flags().GetBool("refresh")

			sessions, err := list.List(lister.ListOptions{
				Config:         config,
				HideAttached:   hideAttached,
				Icons:          icons,
				Json:           jsonOutput,
				Tmux:           tmux,
				Zoxide:         zoxide,
				Tmuxinator:     tmuxinator,
				GitHub:         github,
				HideDuplicates: hideDuplicates,
				Refresh:        refresh,
			})
			if err != nil {
				return fmt.Errorf("couldn't list sessions: %q", err)
			}

			if jsonOutput {
				var sessionsArray []model.SeshSession
				for _, i := range sessions.OrderedIndex {
					sessionsArray = append(sessionsArray, sessions.Directory[i])
				}
				fmt.Println(json.EncodeSessions(sessionsArray))
				return nil
			}

			for _, i := range sessions.OrderedIndex {
				name := sessions.Directory[i].Name
				if icons {
					name = icon.AddIcon(sessions.Directory[i])
				}
				fmt.Println(name)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("config", "c", false, "show configured sessions")
	cmd.Flags().BoolP("json", "j", false, "output as json")
	cmd.Flags().BoolP("tmux", "t", false, "show tmux sessions")
	cmd.Flags().BoolP("zoxide", "z", false, "show zoxide results")
	cmd.Flags().BoolP("hide-attached", "H", false, "don't show currently attached sessions")
	cmd.Flags().BoolP("icons", "i", false, "show icons")
	cmd.Flags().BoolP("tmuxinator", "T", false, "show tmuxinator configs")
	cmd.Flags().BoolP("github", "g", false, "show GitHub organization repositories")
	cmd.Flags().BoolP("hide-duplicates", "d", false, "hide duplicate entries")
	cmd.Flags().BoolP("refresh", "r", false, "force refresh GitHub cache")

	return cmd
}
