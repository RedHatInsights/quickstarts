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

func FindQuickstartById(id string) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

func GetAllQuickstarts(c *gin.Context) {
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
	quickStart, err := FindQuickstartById(c.Param("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quickStart})
}

func deleteQuickstartById(c *gin.Context) {
	quickStart, err := FindQuickstartById(c.Param("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	database.DB.Delete(&quickStart)
	c.JSON(http.StatusOK, gin.H{"msg": "Quickstart successfully removed"})
}

func updateQuickstartById(c *gin.Context) {
	quickStart, err := FindQuickstartById(c.Param("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Not found"})
		return
	}
	if err := c.ShouldBindJSON(&quickStart); err != nil {
		logrus.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	database.DB.Save(quickStart)
	c.JSON(http.StatusOK, gin.H{"data": quickStart})
}

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(subRouter *gin.RouterGroup) {
	subRouter.POST("", createQuickstart)
	subRouter.GET("", GetAllQuickstarts)
	subRouter.GET("/:id", getQuickstartById)
	subRouter.DELETE("/:id", deleteQuickstartById)
	subRouter.PATCH("/:id", updateQuickstartById)
}
