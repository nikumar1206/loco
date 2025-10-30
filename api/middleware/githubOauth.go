package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	json "github.com/goccy/go-json"
	"github.com/nikumar1206/loco/api/models"
	"github.com/patrickmn/go-cache"
)

// cache valid tokens. This cache is actually written to inside the oauth handlers, but read in the middleware
var TokenCache = cache.New(5*time.Minute, 10*time.Minute)

type User struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

type githubAuthInterceptor struct{}

func NewGithubAuthInterceptor() *githubAuthInterceptor {
	return &githubAuthInterceptor{}
}

func (i *githubAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
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

		hc := http.Client{Timeout: 10 * time.Second}
		user := new(User)
		githubReq, err := http.NewRequest("GET", "https://api.github.com/user", nil)
		if err != nil {
			slog.Error(err.Error())
			return nil, connect.NewError(
				connect.CodeInternal,
				err,
			)
		}

		githubReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		githubReq.Header.Add("Accept", "application/vnd.github+json")

		resp, err := hc.Do(githubReq)
		if err != nil {
			slog.Error(err.Error())
			return nil, connect.NewError(
				connect.CodeInternal,
				err,
			)
		}
		defer resp.Body.Close()

		if resp.StatusCode > 299 {
			slog.Error(fmt.Sprintf("received an unexpected status code: %d", resp.StatusCode))
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("Could not confirm identity"))
		}

		err = json.NewDecoder(resp.Body).Decode(user)
		if err != nil {
			slog.Error(err.Error())
			return nil, connect.NewError(
				connect.CodeInternal,
				errors.New("could not decode response from github"),
			)
		}

		// cache the token
		TokenCache.Set(token, user.Login, models.OAuthTokenTTL-(10*time.Minute))

		// inject user login into context
		c := context.WithValue(ctx, "user", user.Login)
		return next(c, req)
	})
}

func (i *githubAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		conn := next(ctx, spec)
		return conn
	})
}

func (i *githubAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		if conn.Spec().Procedure == "/proto.oauth.v1.OAuthService/GithubOAuthDetails" {
			return next(ctx, conn)
		}
		authHeader := conn.RequestHeader().Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return connect.NewError(
				connect.CodeUnauthenticated,
				errors.New("no token provided"),
			)
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if cachedUser, found := TokenCache.Get(token); found {
			usr := cachedUser.(string)
			c := context.WithValue(ctx, "user", usr)
			return next(c, conn)
		}

		hc := http.Client{
			Timeout: 10 * time.Second,
		}
		user := new(User)
		req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
		if err != nil {
			slog.Error(err.Error())
			return connect.NewError(
				connect.CodeInternal,
				err,
			)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Add("Accept", "application/vnd.github+json")

		resp, err := hc.Do(req)
		if err != nil {
			slog.Error(err.Error())
			return connect.NewError(
				connect.CodeInternal,
				err,
			)
		}
		defer resp.Body.Close()

		if resp.StatusCode > 299 {
			slog.Error(fmt.Sprintf("received an unexpected status code: %d", resp.StatusCode))
			return connect.NewError(connect.CodeUnauthenticated, errors.New("Could not confirm identity"))
		}

		err = json.NewDecoder(resp.Body).Decode(user)
		if err != nil {
			slog.Error(err.Error())
			return connect.NewError(
				connect.CodeInternal,
				errors.New("could not decode response from github"),
			)
		}

		// cache the token
		TokenCache.Set(token, user.Login, models.OAuthTokenTTL-(10*time.Minute))

		// inject user login into context
		c := context.WithValue(ctx, "user", user.Login)
		return next(c, conn)
	})
}
