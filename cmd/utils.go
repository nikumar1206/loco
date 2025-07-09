package cmd

import (
	"fmt"
	"os/user"
	"time"

	"github.com/nikumar1206/loco/internal/keychain"
)

func determineHost(isDev bool) string {
	if isDev {
		return "http://localhost:8000"
	} else {
		return "https://loco.deploy-app.com"
	}
}

func getLocoToken() (*keychain.UserToken, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	locoToken, err := keychain.GetGithubToken(usr.Name)
	if err != nil {
		return nil, err
	}

	if locoToken.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		return nil, fmt.Errorf("token is expired or will expire soon. Please re-login via `loco login`")
	}

	return locoToken, err
}
