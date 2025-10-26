package client

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"

	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/pkg/browser"
)

type DeployTokenResponse struct {
	Scopes    []string `json:"scopes"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Registry  string   `json:"registry"`
	Image     string   `json:"image"`
	ExpiresAt string   `json:"expiresAt"`
	Revoked   bool     `json:"revoked"`
	Expired   bool     `json:"expired"`
}

type LoginResponse struct {
	LoginURL  string `json:"loginUrl"`
	SessionID string `json:"sessionId"`
}
type StatusResp struct {
	Token  *string `json:"token"`
	Status string  `json:"status"`
}

func (c *Client) GetDeployToken(locoToken string) (DeployTokenResponse, error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", locoToken),
	}

	resp, err := c.Get("/api/v1/registry/token", headers)
	if err != nil {
		return DeployTokenResponse{}, fmt.Errorf("failed to get deploy token: %v", err)
	}

	var tokenResponse DeployTokenResponse
	if err := json.Unmarshal(resp, &tokenResponse); err != nil {
		return DeployTokenResponse{}, fmt.Errorf("error unmarshalling deploy token response: %v", err)
	}

	return tokenResponse, nil
}

func (c *Client) Login() (string, error) {
	resp, err := c.Get("/api/v1/login", nil)
	if err != nil {
		return "", fmt.Errorf("failed to start login: %v", err)
	}

	var loginResp LoginResponse
	if err = json.Unmarshal(resp, &loginResp); err != nil {
		return "", fmt.Errorf("failed to parse login response: %v", err)
	}

	ok, err := ui.AskYesNo("Loco uses Github OAuth. Do you want to open your browser to authenticate to Loco?")
	if err != nil {
		return "", err
	}

	if !ok {
		return "", fmt.Errorf("cancelled by user")
	}

	err = browser.OpenURL(loginResp.LoginURL)
	if err != nil {
		return "", err
	}

	for {
		statusResp, err := c.checkLoginStatus(loginResp.SessionID)
		if err != nil {
			return "", err
		}

		if statusResp.Status == "completed" {

			if err := keychain.SetGithubToken("", keychain.UserToken{Token: "", ExpiresAt: time.Now()}); err != nil {
				return "", err
			}
			fmt.Println("Login completed!")
			return *statusResp.Token, nil
		}

		time.Sleep(2 * time.Second)
	}
}

func (c *Client) checkLoginStatus(sessionID string) (*StatusResp, error) {
	url := fmt.Sprintf("/api/v1/login/status?sessionId=%s", sessionID)
	resp, err := c.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check login status: %v", err)
	}

	statusResp := new(StatusResp)

	if err := json.Unmarshal(resp, statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %v", err)
	}

	return statusResp, nil
}
