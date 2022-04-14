//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/google/wire"
	"seg-server/internal/biz"
	"seg-server/internal/conf"
	"seg-server/internal/data"
	"seg-server/internal/server"
	"seg-server/internal/service"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, middleware.Middleware, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
