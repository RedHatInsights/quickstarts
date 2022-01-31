package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/routes"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/platform-go-middlewares/request_id"
	"github.com/sirupsen/logrus"
)

func initDependecies() {
	database.Init()
}

func probe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	godotenv.Load()
	config.Init()
	cfg := config.Get()
	initDependecies()
	logrus.WithFields(logrus.Fields{
		"ServerAddr": cfg.ServerAddr,
	})

	r := chi.NewRouter()
	mr := chi.NewRouter()

	r.Use(
		request_id.ConfiguredRequestID("x-rh-insights-request-id"),
		middleware.RealIP,
		middleware.Recoverer,
		middleware.Logger,
	)

	r.Get("/test", probe)
	r.With(routes.PrometheusMiddleware).Route("/api/quickstarts/v1", func(sub chi.Router) {
		sub.Route("/quickstarts", routes.MakeQuickstartsRouter)
		sub.Route("/progress", routes.MakeQuickstartsProgressRouter)
	})
	mr.Get("/", probe)
	mr.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	metricsServer := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler: mr,
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("Metrics server stopped")
		}
	}()

	logrus.Infoln("Starting http server")
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
