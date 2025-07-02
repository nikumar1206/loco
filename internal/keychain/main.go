package keychain

import (
	"encoding/json"
	"time"

	"github.com/zalando/go-keyring"
)

var Service = "loco"

type UserToken struct {
	Token     string
	ExpiresAt time.Time
}

func SetGithubToken(user string, t UserToken) error {
	bytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	return keyring.Set(Service, user, string(bytes))
}

func GetGithubToken(user string) (*UserToken, error) {
	pass, err := keyring.Get(Service, user)
	if err != nil {
		return nil, err
	}
	t := new(UserToken)
	err = json.Unmarshal([]byte(pass), t)
	return t, err
}
