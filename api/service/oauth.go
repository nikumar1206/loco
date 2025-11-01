package service

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nikumar1206/loco/api/models"
	oAuth "github.com/nikumar1206/loco/shared/proto/oauth/v1"
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
