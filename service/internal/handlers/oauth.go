package handlers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	oAuth "github.com/nikumar1206/loco/proto/oauth/v1"
	"github.com/nikumar1206/loco/service/internal/models"
)

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
