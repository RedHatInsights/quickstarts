package routes

import (
	"errors"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getAllQuickstartsProgress(c *gin.Context) {
	var progress []models.QuickstartProgress
	database.DB.Find(&progress)
	c.JSON(http.StatusOK, gin.H{"data": progress})
}

func createQuickstartProgress(c *gin.Context) {
	quickStart, err := FindQuickstartById(c.Param("quickstartId"))
	var progress *models.QuickstartProgress
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	if err := c.ShouldBindJSON(&progress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
	}

	progress.Quickstart = quickStart

	database.DB.Create(&progress)
	c.JSON(http.StatusOK, gin.H{"id": progress.ID})
}

func MakeQuickstartsProgressRouter(subRouter *gin.RouterGroup) {
	subRouter.GET("", getAllQuickstartsProgress)
	subRouter.POST("/:quickstartId", createQuickstartProgress)
}
