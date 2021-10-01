package routes

import (
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func MakeQuickstartsRouter(subRouter *gin.RouterGroup) {
	subRouter.POST("", func(c *gin.Context) {
		var quickStart *models.Quickstart
		if err := c.ShouldBindJSON(&quickStart); err != nil {
			logrus.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		}

		database.DB.Create(&quickStart)
		c.JSON(http.StatusOK, gin.H{"id": quickStart.ID})
	})

	subRouter.GET("", func(c *gin.Context) {
		var quickStarts []models.Quickstart
		database.DB.Find(&quickStarts)
		c.JSON(http.StatusOK, gin.H{"data": quickStarts})
	})
}
