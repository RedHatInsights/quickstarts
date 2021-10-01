package routes

import (
	"errors"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func getAllQuickstarts(c *gin.Context) {
	var quickStarts []models.Quickstart
	database.DB.Find(&quickStarts)
	c.JSON(http.StatusOK, gin.H{"data": quickStarts})
}

func createQuickstart(c *gin.Context) {
	var quickStart *models.Quickstart
	if err := c.ShouldBindJSON(&quickStart); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
	}

	database.DB.Create(&quickStart)
	c.JSON(http.StatusOK, gin.H{"id": quickStart.ID})
}

func getQuickstartById(c *gin.Context) {
	var quickstart models.Quickstart
	quickstartId := c.Param("id")
	err := database.DB.First(&quickstart, quickstartId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quickstart})

}

func deleteQuickstartById(c *gin.Context) {
	var quickstart models.Quickstart
	quickstartId := c.Param("id")
	err := database.DB.First(&quickstart, quickstartId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	database.DB.Delete(&quickstart)
	c.JSON(http.StatusOK, gin.H{"msg": "Quickstart successfully removed"})
}

func updateQuickstartById(c *gin.Context) {
	var quickstart models.Quickstart
	quickstartId := c.Param("id")
	err := database.DB.First(&quickstart, quickstartId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	if err := c.ShouldBindJSON(&quickstart); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	database.DB.Save(quickstart)
	c.JSON(http.StatusOK, gin.H{"data": quickstart})
}

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(subRouter *gin.RouterGroup) {
	subRouter.POST("", createQuickstart)
	subRouter.GET("", getAllQuickstarts)
	subRouter.GET("/:id", getQuickstartById)
	subRouter.DELETE("/:id", deleteQuickstartById)
	subRouter.PATCH("/:id", updateQuickstartById)
}
