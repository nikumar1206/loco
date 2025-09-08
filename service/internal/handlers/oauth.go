package handlers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/gofiber/fiber/v3"
	oAuth "github.com/nikumar1206/loco/proto/oauth/v1"
	"github.com/nikumar1206/loco/service/internal/models"
)

// BuildOauthRouter houses APIs for interacting with OAuth services. currently github, but can be google one day
func BuildOauthRouter(app *fiber.App, appConfig *models.AppConfig) {
	githubOAuthGroup := app.Group("/api/v1/oauth/github")

	githubOAuthGroup.Get("", getTokenDetails())
}

func getTokenDetails() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"clientId": models.OAuthConf.ClientID,
			"tokenTTL": models.OAuthTokenTTL.Seconds(),
		})
	}
}

type OAuthServer struct{}

func (s *OAuthServer) GithubOAuthDetails(
	ctx context.Context, req *connect.Request[oAuth.GithubOAuthDetailsRequest],
) (*connect.Response[oAuth.GithubOAuthDetailsResponse], error) {
	slog.InfoContext(ctx, "Request headers: ", slog.Any("headers", req.Header()))
	res := connect.NewResponse(&oAuth.GithubOAuthDetailsResponse{
		ClientId: models.OAuthConf.ClientID,
		TokenTtl: models.OAuthTokenTTL.Seconds(),
	})
	return res, nil
}
