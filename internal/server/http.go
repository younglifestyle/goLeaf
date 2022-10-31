package server

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	netHttp "net/http"
	v1 "seg-server/api/leaf-grpc/v1"
	"seg-server/internal/conf"
	"seg-server/internal/service"
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

	//r := srv.Route("/")
	//r.POST("/login", func(c http.Context) error {
	//	c.JSON(200, "")
	//
	//	return nil
	//})
	//
	//srv.HandlePrefix("/web/",
	//	netHttp.StripPrefix("/web/",
	//		netHttp.FileServer(netHttp.Dir("./web"))))
	//srv.HandlePrefix("/pages",
	//	netHttp.StripPrefix("/pages",
	//		netHttp.FileServer(netHttp.Dir("./web/pages"))))

	r := gin.Default()

	r.LoadHTMLFiles("./web/login.html")

	//r.Static("/sdk", "./web/sdk")
	r.Static("/web", "./web")
	r.Static("/pages", "./web/pages")
	r.GET("/login", func(c *gin.Context) {
		c.HTML(netHttp.StatusOK, "login.html", "")
	})
	r.POST("/login", func(c *gin.Context) {
		c.JSON(netHttp.StatusOK, gin.H{
			"status": 0,
			"msg":    "",
			"data":   "",
		})
	})
	srv.HandlePrefix("/", r)

	return srv
}
