package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func getAllQuickstartsProgress(c *gin.Context) {
	var progress []models.QuickstartProgress
	database.DB.Find(&progress)
	c.JSON(http.StatusOK, gin.H{"data": progress})
}

func getQuickstartProgress(c *gin.Context) {
	queries := c.Request.URL.Query()
	logrus.Info(queries)

	var accountId int
	var quickstartId int
	accountId, _ = strconv.Atoi(queries.Get("account"))
	quickstartId, _ = strconv.Atoi(queries.Get("quickstart"))

	if accountId != 0 || quickstartId != 0 {
		var where models.QuickstartProgress
		var progresses []models.QuickstartProgress
		if accountId != 0 {
			where.AccountId = accountId
		}

		if quickstartId != 0 {
			where.QuickstartID = uint(quickstartId)
		}
		database.DB.Where(where).Find(&progresses)
		c.JSON(http.StatusOK, gin.H{"data": progresses})
		return
	} else {
		getAllQuickstartsProgress(c)
	}

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
	subRouter.GET("", getQuickstartProgress)
	subRouter.POST("/:quickstartId", createQuickstartProgress)
}
