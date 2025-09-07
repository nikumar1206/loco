package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
	"github.com/patrickmn/go-cache"
)

// cache valid tokens. This cache is actually written to inside the oauth handlers, but read in the middleware
var TokenCache = cache.New(5*time.Minute, 10*time.Minute)

type User struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

func GithubTokenValidator() fiber.Handler {
	return func(c fiber.Ctx) error {
		// skip auth for these endpoints
		if c.Path() == "/api/v1/oauth/github" {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return utils.SendErrorResponse(c, http.StatusUnauthorized, "Not Authorized")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		cachedUser, found := TokenCache.Get(token)
		if found {
			usr := cachedUser.(string)
			c.Locals("user", usr)
			return c.Next()
		}

		user := new(User)

		resp, err := client.Resty.R().
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
			SetHeader("Accept", "application/vnd.github+json").
			SetResult(&user).
			Get("https://api.github.com/user")
		if err != nil {
			slog.Error(err.Error())
			return err
		}

		if resp.IsError() {
			slog.Error(resp.String())
			return utils.SendErrorResponse(c, http.StatusUnauthorized, "Could not confirm identity")
		}

		TokenCache.Set(token, user.Login, models.OAuthTokenTTL-(10*time.Minute))
		c.Locals("user", user.Login)

		return c.Next()
	}
}
