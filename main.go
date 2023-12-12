package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/logger"
	"github.com/RedHatInsights/quickstarts/pkg/routes"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
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

func specHandler(w http.ResponseWriter, r *http.Request) {
	root := "./spec"
	fs := http.FileServer(http.Dir(root))
	if _, err := os.Stat(root + r.RequestURI); os.IsNotExist(err) {
		http.StripPrefix(r.RequestURI, fs).ServeHTTP(w, r)
	} else {
		fs.ServeHTTP(w, r)
	}
}

func main() {
	godotenv.Load()
	config.Init()
	cfg := config.Get()
	initDependecies()
	setupGlobalLogger(cfg)
	logrus.WithFields(logrus.Fields{
		"ServerAddr": cfg.ServerAddr,
	})

	r := chi.NewRouter()
	mr := chi.NewRouter()

	routerLogger := logrus.New()

	r.Use(
		request_id.ConfiguredRequestID("x-rh-insights-request-id"),
		middleware.RealIP,
		middleware.Recoverer,
		middleware.RequestLogger(logger.NewLogger(cfg, routerLogger)),
	)

	root := "./spec/"
	fs := http.FileServer(http.Dir(root))
	r.With(routes.PrometheusMiddleware).Route("/api/quickstarts/v1", func(sub chi.Router) {
		sub.Route("/quickstarts", routes.MakeQuickstartsRouter)
		sub.Route("/progress", routes.MakeQuickstartsProgressRouter)
		sub.Route("/helptopics", routes.MakeHelpTopicsRouter)
		sub.Route("/favorites", routes.MakeFavoriteQuickstartsRouter)
		sub.Handle("/spec/*", http.StripPrefix("/api/quickstarts/v1/spec", fs))
	})
	mr.Get("/", probe)
	mr.Handle("/metrics", promhttp.Handler())
	r.Get("/test", probe)

	// fmt.Println(cfg.ServerAddr)
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

func setupGlobalLogger(opts *config.QuickstartsConfig) {
	logLevel, err := logrus.ParseLevel(opts.LogLevel)
	if err != nil {
		logLevel = logrus.ErrorLevel
	}
	logrus.SetLevel(logLevel)
}
