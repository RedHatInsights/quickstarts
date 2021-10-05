package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func initDependecies() {
	database.Init()
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func setupRouter(cfg *config.QuickstartsConfig) *gin.Engine {
	engine := gin.Default()
	engine.GET("/test", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"message": "This is a test response",
		})
	})

	engine.GET("/api/quickstarts/v1/openapi.json", func(c *gin.Context) {
		c.File(cfg.OpenApiSpecPath)
	})

	versionGroup := engine.Group("/api/quickstarts/v1")
	quickstartsGroup := versionGroup.Group("/quickstarts")
	quickstartsProgressGroup := versionGroup.Group("/progress")
	routes.MakeQuickstartsRouter(quickstartsGroup)
	routes.MakeQuickstartsProgressRouter(quickstartsProgressGroup)

	return engine
}

func main() {
	godotenv.Load()
	config.Init()
	cfg := config.Get()
	initDependecies()
	logrus.WithFields(logrus.Fields{
		"ServerAddr": cfg.ServerAddr,
	})

	// done := make(chan struct{})
	// sigint := make(chan os.Signal, 1)
	// signal.Notify(sigint)

	engine := setupRouter(cfg)

	server := http.Server{
		Addr:    cfg.ServerAddr,
		Handler: engine,
	}

	metricsEngine := gin.Default()
	metricsEngine.GET("/", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"message": "OK",
		})
	})

	/**Find a handle for all http request types*/
	metricsEngine.GET("/metrics", prometheusHandler())
	metricsEngine.POST("/metrics", prometheusHandler())
	metricsEngine.PUT("/metrics", prometheusHandler())
	metricsEngine.PATCH("/metrics", prometheusHandler())
	metricsEngine.DELETE("/metrics", prometheusHandler())

	metricsServer := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler: metricsEngine,
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("Metrics server stopped")
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal("Api server has stopped")
	}

	// <-done
	// logrus.Info("Gracefully stopping server")

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// defer func() {
	// 	// extra handling here
	// 	cancel()
	// }()

	// if err := server.Shutdown(ctx); err != nil {
	// 	logrus.Fatal("Server shutdown failed:%+v", err)
	// }
	// logrus.Info("Server stypped properly")
}
