package service

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/nikumar1206/loco/api/client"
	"github.com/nikumar1206/loco/api/models"
	registry "github.com/nikumar1206/loco/shared/proto/registry/v1"
)

type RegistryServer struct {
	AppConfig models.AppConfig
}

func (s *RegistryServer) GitlabToken(
	ctx context.Context, req *connect.Request[registry.GitlabTokenRequest],
) (*connect.Response[registry.GitlabTokenResponse], error) {
	projectId := s.AppConfig.ProjectID
	tokenName := s.AppConfig.DeployTokenName

	expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")

	payload := map[string]any{
		"name":       tokenName,
		"scopes":     []string{"write_registry", "read_registry"},
		"expires_at": expiry,
	}

	gitlabResp, err := client.NewClient(s.AppConfig.GitlabURL).GetDeployToken(ctx, s.AppConfig.GitlabPAT, projectId, payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	res := connect.NewResponse(&registry.GitlabTokenResponse{
		Username:  gitlabResp.Username,
		Token:     gitlabResp.Token,
		Registry:  "registry.gitlab.com",
		Image:     "registry.gitlab.com/locomotive-group/loco-ecr",
		ExpiresAt: gitlabResp.ExpiresAt,
		Revoked:   gitlabResp.Revoked,
		Expired:   gitlabResp.Expired,
		Scopes:    gitlabResp.Scopes,
	})
	return res, nil
}
