package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func initDependecies() {

}

func main() {
	godotenv.Load()
	initDependecies()
	godotenv.Load()
	cfg := config.Get()
	logrus.WithFields(logrus.Fields{
		"ServerAddr": cfg.ServerAddr,
	})

	dbdns := "host=localhost user=martin password=martin dbname=quickstarts port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dbdns), &gorm.Config{})

	if db.Migrator().HasTable(&models.Quickstart{}) == false {
		db.Migrator().CreateTable(&models.Quickstart{})
	}

	if err != nil {
		logrus.Error(err)
		panic("failed to connect database")
	}

	// done := make(chan struct{})
	// sigint := make(chan os.Signal, 1)
	// signal.Notify(sigint)

	engine := gin.Default()
	engine.GET("/test", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"message": "This is a test response",
		})
	})

	engine.GET("/api/quickstarts/v1/openapi.json", func(c *gin.Context) {
		c.File(cfg.OpenApiSpecPath)
	})

	engine.POST("/api/quickstarts/v1/quickstarts", func(c *gin.Context) {
		var quickStart *models.Quickstart
		if err := c.ShouldBindJSON(&quickStart); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		}

		db.Create(&quickStart)
		fmt.Println("After create")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		}
		c.JSON(http.StatusOK, gin.H{"id": quickStart.ID})
	})

	engine.GET("/api/quickstarts/v1/quickstarts", func(c *gin.Context) {
		var quickStarts []models.Quickstart
		db.Find(&quickStarts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		}
		c.JSON(http.StatusOK, gin.H{"data": quickStarts})
	})

	server := http.Server{
		Addr:    cfg.ServerAddr,
		Handler: engine,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal("listen: %s\n", err)
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
