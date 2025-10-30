package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/user"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/nikumar1206/loco/internal/ui"
	oAuth "github.com/nikumar1206/loco/proto/oauth/v1"
	"github.com/nikumar1206/loco/proto/oauth/v1/oauthv1connect"
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

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to loco via Github OAuth",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getHost(cmd)
		if err != nil {
			return err
		}
		user, err := user.Current()
		if err != nil {
			slog.Debug("failed to get current user", "error", err)
			return err
		}
		t, err := keychain.GetGithubToken(user.Name)
		if err == nil {
			if !t.ExpiresAt.Before(time.Now().Add(1 * time.Hour)) {
				checkmark := lipgloss.NewStyle().Foreground(ui.LocoGreen).Render("✔")
				message := lipgloss.NewStyle().Bold(true).Foreground(ui.LocoOrange).Render("Already logged in!")
				subtext := lipgloss.NewStyle().
					Foreground(ui.LocoLightGray).
					Render("You can continue using loco")

				fmt.Printf("%s %s\n%s\n", checkmark, message, subtext)
				return nil
			}
			slog.Debug("token is expired or will expire soon", "expires_at", t.ExpiresAt)
		} else {
			slog.Debug("no token found in keychain", "error", err)
		}
		c := client.NewClient("https://github.com")

		oAuthClient := oauthv1connect.NewOAuthServiceClient(&http.Client{}, host)
		resp, err := oAuthClient.GithubOAuthDetails(context.Background(), connect.NewRequest(&oAuth.GithubOAuthDetailsRequest{}))
		if err != nil {
			slog.Debug("failed to get oauth details", "error", err)
			return err
		}
		slog.Debug("retrieved oauth details", "client_id", resp.Msg.ClientId)

		payload := DeviceCodeRequest{
			ClientId: resp.Msg.ClientId,
			Scope:    "read:user",
		}

		req, err := c.Post("/login/device/code", payload, map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
		if err != nil {
			slog.Debug("failed to get device code", "error", err)
			return err
		}

		deviceTokenResponse := new(DeviceCodeResponse)
		err = json.Unmarshal(req, deviceTokenResponse)
		if err != nil {
			slog.Debug("failed to unmarshal device code response", "error", err)
			return err
		}

		tokenChan := make(chan AuthTokenResponse, 1)
		errorChan := make(chan error, 1)

		go func() {
			pollErr := pollAuthToken(c, payload.ClientId, deviceTokenResponse.DeviceCode, deviceTokenResponse.Interval, tokenChan)
			if pollErr != nil {
				fmt.Println(pollErr.Error())
				errorChan <- pollErr
			}
		}()

		m := initialModel(deviceTokenResponse.UserCode, deviceTokenResponse.VerificationURI, tokenChan, errorChan)
		p := tea.NewProgram(m)

		fm, err := p.Run()
		if err != nil {
			return err
		}

		finalM, ok := fm.(model)
		if !ok {
			return fmt.Errorf("%w: unexpected model type", ErrCommandFailed)
		}

		if finalM.err != nil {
			return finalM.err
		}
		if finalM.tokenResp != nil {
			if err := keychain.SetGithubToken(user.Name, keychain.UserToken{
				Token: finalM.tokenResp.AccessToken,
				// subtract 10 mins?
				ExpiresAt: time.Now().Add(time.Duration(resp.Msg.TokenTtl)*time.Second - (10 * time.Minute)),
			}); err != nil {
				return fmt.Errorf("%w: %w", ErrAuthFailed, err)
			}
		}

		return nil
	},
}

func pollAuthToken(c *client.Client, clientId string, deviceCode string, interval int, tokenChan chan AuthTokenResponse) error {
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
			if apiError, ok := err.(*client.APIError); ok {
				switch apiError.StatusCode {
				case 400:
					slog.Debug("authorization pending", "status_code", apiError.StatusCode)
					time.Sleep(time.Duration(interval) * time.Second)
					continue
				case 403: // rate limit or access denied
					slog.Debug("access denied or rate limited", "status_code", apiError.StatusCode, "error", err)
					return fmt.Errorf("access denied or rate limited: %w", err)
				default:
					slog.Debug("API error while polling for token", "status_code", apiError.StatusCode, "error", err)
					return fmt.Errorf("API error: %w", err)
				}
			} else {
				slog.Debug("network error while polling for token", "error", err)
				return fmt.Errorf("network error: %w", err)
			}
		}

		authTokenResponse := new(AuthTokenResponse)
		err = json.Unmarshal(resp, authTokenResponse)
		if err != nil {
			slog.Debug("failed to unmarshal auth token response", "error", err)
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
	loadingFrames   []string
	userCode        string
	verificationURI string
	err             error
	tokenChan       <-chan AuthTokenResponse
	errorChan       <-chan error
	tokenResp       *AuthTokenResponse
	frameIndex      int
	polling         bool
	done            bool
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
			errorStyle := lipgloss.NewStyle().Foreground(ui.LocoRed).Bold(true)
			return fmt.Sprintf("%s\n%s\n",
				errorStyle.Render("Authentication failed:"),
				lipgloss.NewStyle().Foreground(ui.LocoDarkGray).Render(m.err.Error()))
		}
		return lipgloss.NewStyle().Foreground(ui.LocoOrange).Bold(true).Render("✓ Authentication successful!") + "\n"
	}

	codeStyle := lipgloss.NewStyle().Foreground(ui.LocoOrange).Bold(true).Padding(0, 0)
	urlStyle := lipgloss.NewStyle().Foreground(ui.LocoOrange).Underline(true)
	instructionStyle := lipgloss.NewStyle().Foreground(ui.LocoLightGray)
	spinnerStyle := lipgloss.NewStyle().Foreground(ui.LocoOrange).Bold(true)

	spinner := ""
	if len(m.loadingFrames) > 0 {
		spinner = spinnerStyle.Render(m.loadingFrames[m.frameIndex])
	}

	return fmt.Sprintf(
		"%s %s\n\n%s %s\n\n%s %s\n\n%s",
		instructionStyle.Render("Please open the following URL in your browser:"),
		urlStyle.Render(m.verificationURI),
		instructionStyle.Render("Then, enter the following user code:"),
		codeStyle.Render(m.userCode),
		spinner,
		instructionStyle.Render("Waiting for authentication..."),
		lipgloss.NewStyle().Foreground(ui.LocoLightGray).Faint(true).Render("Press 'q' or Ctrl+C to quit"),
	)
}
