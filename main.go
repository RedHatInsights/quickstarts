package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/gin-gonic/gin"
	redoc "github.com/go-openapi/runtime/middleware"
	"github.com/sirupsen/logrus"
)

func setupDocsMiddleware(handler http.Handler) http.Handler {
	conf := redoc.RedocOpts{
		SpecURL: "/api/quickstarts/v1/openapi.json",
	}
	return redoc.Redoc(conf, handler)
}

func initDependecies() {

}

func main() {
	initDependecies()
	cfg := config.Get()
	logrus.WithFields(logrus.Fields{
		"ServerAddr": cfg.ServerAddr,
	})

	done := make(chan struct{})
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint)

	engine := gin.Default()
	engine.GET("/test", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"message": "This is a test response",
		})
	})

	server := http.Server{
		Addr:    cfg.ServerAddr,
		Handler: engine,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal("listen: %s\n", err)
	}

	<-done
	logrus.Info("Gracefully stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer func() {
		// extra handling here
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Server shutdown failed:%+v", err)
	}
	logrus.Info("Server stypped properly")
}
