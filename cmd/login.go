package cmd

import (
	"encoding/json"
	"fmt"
	"os/user"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/api"
	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/spf13/cobra"
)

type DeviceCodeRequest struct {
	ClientId string `json:"client_id"`
	Scope    string `json:"scope"`
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type AuthTokenRequest struct {
	ClientId   string `json:"client_id"`
	DeviceCode string `json:"device_code"`
	GrantType  string `json:"grant_type"`
}

type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type TokenDetails struct {
	ClientId string  `json:"clientId"`
	TokenTTL float64 `json:"tokenTTL"`
}

var testCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to loco via Github OAuth",
	RunE: func(cmd *cobra.Command, args []string) error {
		user, err := user.Current()
		if err != nil {
			return err
		}

		t, err := keychain.GetGithubToken(user.Name)
		if err != nil {
			return err
		}

		if !t.ExpiresAt.Before(time.Now().Add(1 * time.Hour)) {
			locoGreen := lipgloss.Color("#10B981")
			locoGray := lipgloss.Color("#9CA3AF")

			checkmark := lipgloss.NewStyle().Foreground(locoGreen).Render("✔")
			message := lipgloss.NewStyle().Foreground(orange).Render("Already logged in!")
			subtext := lipgloss.NewStyle().
				Foreground(locoGray).
				Render("You can continue using loco")

			fmt.Printf("%s %s\n%s\n", checkmark, message, subtext)
			return nil
		}

		c := api.NewClient("https://github.com")

		isDev, err := cmd.Flags().GetBool("dev")
		if err != nil {
			return fmt.Errorf("error reading dev flag: %w", err)
		}

		var host string
		if isDev {
			host = "http://localhost:8000"
		} else {
			host = "https://loco.deploy-app.com"
		}

		locoClient := api.NewClient(host)

		res, err := locoClient.Get("/api/v1/oauth/github", nil)
		if err != nil {
			return err
		}
		tokenDetails := new(TokenDetails)
		err = json.Unmarshal(res, tokenDetails)
		if err != nil {
			return err
		}

		payload := DeviceCodeRequest{
			ClientId: tokenDetails.ClientId,
			Scope:    "read:user",
		}

		req, err := c.Post("/login/device/code", payload, map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
		if err != nil {
			return err
		}

		deviceTokenResponse := new(DeviceCodeResponse)
		err = json.Unmarshal(req, deviceTokenResponse)
		if err != nil {
			return err
		}

		tokenChan := make(chan AuthTokenResponse, 1)
		errorChan := make(chan error, 1)

		go func() {
			err := pollAuthToken(c, payload.ClientId, deviceTokenResponse.DeviceCode, deviceTokenResponse.Interval, tokenChan)
			if err != nil {
				errorChan <- err
			}
		}()

		m := initialModel(deviceTokenResponse.UserCode, deviceTokenResponse.VerificationURI, tokenChan, errorChan)
		p := tea.NewProgram(m)

		fm, err := p.Run()
		if err != nil {
			return err
		}

		finalM := fm.(model)

		if finalM.err != nil {
			return finalM.err
		}
		if finalM.tokenResp != nil {
			keychain.SetGithubToken(user.Name, keychain.UserToken{
				Token: finalM.tokenResp.AccessToken,
				// subtract 10 mins?
				ExpiresAt: time.Now().Add(time.Duration(tokenDetails.TokenTTL)*time.Second - (10 * time.Minute)),
			})
		}

		return nil
	},
}

func pollAuthToken(c *api.Client, clientId string, deviceCode string, interval int, tokenChan chan AuthTokenResponse) error {
	authTokenRequest := AuthTokenRequest{
		ClientId:   clientId,
		DeviceCode: deviceCode,
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
	}

	for {
		resp, err := c.Post("/login/oauth/access_token", authTokenRequest, map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
		if err != nil {
			if apiError, ok := err.(*api.APIError); ok {
				switch apiError.StatusCode {
				case 400:
					time.Sleep(time.Duration(interval) * time.Second)
					continue
				case 403: // rate limit or access denied
					return fmt.Errorf("access denied or rate limited: %w", err)
				default:
					return fmt.Errorf("API error: %w", err)
				}
			} else {
				return fmt.Errorf("network error: %w", err)
			}
		}

		authTokenResponse := new(AuthTokenResponse)
		err = json.Unmarshal(resp, authTokenResponse)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if authTokenResponse.AccessToken != "" {
			tokenChan <- *authTokenResponse
			break
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}

	return nil
}

var (
	orange   = lipgloss.Color("#FF6F00")
	softGray = lipgloss.Color("#B0BEC5")
	black    = lipgloss.Color("#212121")
	white    = lipgloss.Color("#FFFFFF")
	red      = lipgloss.Color("#F44336")
)

type (
	tickMsg        time.Time
	authSuccessMsg struct {
		Token AuthTokenResponse
	}
	authErrorMsg struct {
		Error error
	}
)

func waitForToken(tokenChan <-chan AuthTokenResponse) tea.Cmd {
	return func() tea.Msg {
		token := <-tokenChan
		return authSuccessMsg{Token: token}
	}
}

func waitForError(errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-errorChan
		return authErrorMsg{Error: err}
	}
}

type model struct {
	userCode        string
	verificationURI string
	loadingFrames   []string
	frameIndex      int
	polling         bool
	done            bool
	err             error
	tokenChan       <-chan AuthTokenResponse
	errorChan       <-chan error
	tokenResp       *AuthTokenResponse
}

func initialModel(userCode string, verificationUri string, tokenChan <-chan AuthTokenResponse, errorChan <-chan error) model {
	return model{
		userCode:        userCode,
		verificationURI: verificationUri,
		loadingFrames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		frameIndex:      0,
		polling:         true,
		done:            false,
		tokenChan:       tokenChan,
		errorChan:       errorChan,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		waitForToken(m.tokenChan),
		waitForError(m.errorChan),
	)
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		if m.polling && !m.done {
			m.frameIndex = (m.frameIndex + 1) % len(m.loadingFrames)
			return m, tick()
		}
	case authSuccessMsg:
		m.polling = false
		m.done = true
		m.tokenResp = &msg.Token
		return m, tea.Quit
	case authErrorMsg:
		m.polling = false
		m.done = true
		m.err = msg.Error
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	if m.done {
		if m.err != nil {
			errorStyle := lipgloss.NewStyle().Foreground(red).Bold(true)
			return fmt.Sprintf("%s\n%s\n",
				errorStyle.Render("Authentication failed:"),
				lipgloss.NewStyle().Foreground(softGray).Render(m.err.Error()))
		}
		return lipgloss.NewStyle().Foreground(orange).Bold(true).Render("✓ Authentication successful!") + "\n"
	}

	codeStyle := lipgloss.NewStyle().Foreground(orange).Bold(true).Padding(0, 0)
	urlStyle := lipgloss.NewStyle().Foreground(orange).Underline(true)
	instructionStyle := lipgloss.NewStyle().Foreground(softGray)
	spinnerStyle := lipgloss.NewStyle().Foreground(orange).Bold(true)

	spinner := ""
	if len(m.loadingFrames) > 0 {
		spinner = spinnerStyle.Render(m.loadingFrames[m.frameIndex])
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n\n%s %s\n\n%s",
		instructionStyle.Render("Please open the following URL in your browser:"),
		urlStyle.Render(m.verificationURI),
		instructionStyle.Render("Then, enter the following user code:"),
		codeStyle.Render(m.userCode),
		spinner,
		instructionStyle.Render("Waiting for authentication..."),
		lipgloss.NewStyle().Foreground(softGray).Faint(true).Render("Press 'q' or Ctrl+C to quit"),
	)
}
