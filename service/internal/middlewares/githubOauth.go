package middlewares

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/patrickmn/go-cache"
)

// cache valid tokens. This cache is actually written to inside the oauth handlers, but read in the middleware
var TokenCache = cache.New(5*time.Minute, 10*time.Minute)

type User struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

func GithubTokenValidator() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if req.Spec().Procedure == "/proto.oauth.v1.OAuthService/GithubOAuthDetails" {
				return next(ctx, req)
			}

			authHeader := req.Header().Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("no token provided"),
				)
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			if cachedUser, found := TokenCache.Get(token); found {
				usr := cachedUser.(string)
				c := context.WithValue(ctx, "user", usr)
				return next(c, req)
			}

			user := new(User)
			resp, err := client.Resty.R().
				SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
				SetHeader("Accept", "application/vnd.github+json").
				SetResult(user).
				Get("https://api.github.com/user")
			if err != nil {
				slog.Error(err.Error())
				return nil, connect.NewError(
					connect.CodeInternal,
					err,
				)
			}

			if resp.IsError() {
				slog.Error(resp.String())
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("Could not confirm identity"))
			}

			// cache the token
			TokenCache.Set(token, user.Login, models.OAuthTokenTTL-(10*time.Minute))

			// inject user login into context
			c := context.WithValue(ctx, "user", user.Login)
			return next(c, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
