package client

import "errors"

var (
	ErrDeploymentNotFound = errors.New("deployment not found")
	ErrNamespaceNotFound  = errors.New("namespace not found")
	ErrServiceNotFound    = errors.New("service not found")
	ErrSecretNotFound     = errors.New("secret not found")
	ErrHTTPRouteNotFound  = errors.New("HTTP route not found")
)
