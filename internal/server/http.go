package server

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "goLeaf/api/leaf-grpc/v1"
	"goLeaf/internal/conf"
	"goLeaf/internal/service"
)

// NewHTTPServer new a HTTP server.
func NewHTTPServer(c *conf.Server, idGenService *service.IdGenService, metricMidSrv middleware.Middleware, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			metricMidSrv,
		),
		http.Filter(handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST"}),
		)),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle("/metrics", promhttp.Handler())
	v1.RegisterLeafSegmentServiceHTTPServer(srv, idGenService)
	v1.RegisterLeafSnowflakeServiceHTTPServer(srv, idGenService)

	r := gin.Default()
	r.GET("/api/segment/get/:tag", idGenService.GetSegmentID)
	r.GET("/api/snowflake/get", idGenService.GetSnowflakeID)
	r.Static("/web", "./web")
	r.Static("/pages", "./web/pages")
	srv.HandlePrefix("/", r)

	return srv
}
