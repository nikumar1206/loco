package loco

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/user"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(whoamiCmd)
}

type GithubUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "displays information on the logged in user",
	RunE: func(cmd *cobra.Command, args []string) error {
		user, err := user.Current()
		if err != nil {
			slog.Debug("failed to get current user", "error", err)
			return err
		}
		t, err := keychain.GetGithubToken(user.Name)
		if err != nil {
			return ErrLoginRequired
		}

		c := client.NewClient("https://api.github.com")
		req, err := c.Get("/user", map[string]string{
			"Accept":               "application/vnd.github+json",
			"Authorization":        fmt.Sprintf("Bearer %s", t.Token),
			"X-GitHub-Api-Version": "2022-11-28",
		})
		if err != nil {
			slog.Debug("failed to get user info", "error", err)
			return fmt.Errorf("failed to get user info. Try logging in again via `loco login`")
		}

		githubUser := new(GithubUser)
		err = json.Unmarshal(req, githubUser)
		if err != nil {
			slog.Debug("failed to unmarshal user info", "error", err)
			return err
		}

		name := lipgloss.NewStyle().Bold(true).Foreground(ui.LocoOrange).Render(githubUser.Name)

		fmt.Printf("Logged in as %s 👋\n", name)

		return nil
	},
}
