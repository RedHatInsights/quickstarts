package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gitconfig "github.com/RedHatInsights/quickstarts/pkg/git-service/config"
	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	githandlers "github.com/RedHatInsights/quickstarts/pkg/git-service/handlers"
	pskmw "github.com/RedHatInsights/quickstarts/pkg/git-service/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()
	gitconfig.Init()
	cfg := gitconfig.Get()

	disabled := os.Getenv("GIT_SERVICE_ENABLED") != "true"

	if disabled && !clowder.IsClowderEnabled() {
		logrus.Warn("GIT_SERVICE_ENABLED is not set; assuming local development mode")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	if disabled {
		logrus.Info("GIT_SERVICE_ENABLED is not set, running in health-only mode")
	} else if cfg.PSKToken == "" || cfg.GitHubToken == "" {
		logrus.Warn("GIT_SERVICE_ENABLED is true but secrets not available, running in health-only mode")
	} else {
		repoMgr, err := gitops.InitRepo(cfg.RepoURL, cfg.RepoPath, cfg.GitHubToken, cfg.BaseBranch)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to initialize repository")
		}

		ghClient, err := ghclient.NewClient(cfg.GitHubToken, cfg.RepoURL)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to initialize GitHub client")
		}

		handler := githandlers.NewHandler(repoMgr, ghClient, cfg.ReviewersTeam, cfg.QuickstartsDirPath)

		r.Route("/api/v1", func(r chi.Router) {
			r.Use(pskmw.PSKAuth(cfg.PSKToken))
			r.Post("/submit-pr", handler.SubmitPR)
			r.Get("/list-quickstarts", handler.ListQuickstarts)
			r.Get("/quickstart-content/{name}", handler.GetQuickstartContent)
		})
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logrus.WithField("port", cfg.Port).Info("Starting quickstarts-git-service")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Server failed to start")
		}
	}()

	<-done
	logrus.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server shutdown error")
	}
}
