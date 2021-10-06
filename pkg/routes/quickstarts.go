package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/gin-gonic/gin"
)

func FindQuickstartById(id int) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

func GetAllQuickstarts(c *gin.Context) {
	var quickStarts []models.Quickstart
	var bundlesQuery, bundlesExists = c.GetQueryArray("[]bundles")
	var bundleQuery, bundleExists = c.GetQuery("bundle")

	// Look for gorm supported APi instead of using RAW query
	// sample query /api/quickstarts/v1/quickstarts?[]bundles=settings&[]bundles=insights
	if bundlesExists {
		var conditions []string
		for _, s := range bundlesQuery {
			conditions = append(conditions, fmt.Sprintf("(bundles)::jsonb ? '%s'", s))
		}
		where := strings.Join(conditions, "OR ")
		database.DB.Raw(fmt.Sprintf("SELECT * FROM quickstarts WHERE %s", where)).Scan(&quickStarts)
	} else if bundleExists {
		database.DB.Raw(fmt.Sprintf("SELECT * FROM quickstarts WHERE (bundles)::jsonb ? '%s'", bundleQuery)).Scan(&quickStarts)
	} else {
		database.DB.Find(&quickStarts)
	}
	c.JSON(http.StatusOK, gin.H{"data": quickStarts})
}

func createQuickstart(c *gin.Context) {
	var quickStart *models.Quickstart
	if err := c.ShouldBindJSON(&quickStart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		c.Abort()
		return
	}

	database.DB.Create(&quickStart)
	c.JSON(http.StatusOK, gin.H{"id": quickStart.ID})
}

func getQuickstartById(c *gin.Context) {
	quickStart, _ := c.Get("quickstart")
	c.JSON(http.StatusOK, gin.H{"data": quickStart})
}

func deleteQuickstartById(c *gin.Context) {
	quickStart, _ := c.Get("quickstart")
	err := database.DB.Delete(quickStart).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "Quickstart successfully removed"})
}

func updateQuickstartById(c *gin.Context) {
	quickStart, _ := c.Get("quickstart")
	if err := c.ShouldBindJSON(&quickStart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		c.Abort()
		return
	}
	err := database.DB.Save(quickStart).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quickStart})
}

func entityContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		if quickstartId := c.Param("id"); quickstartId != "" {
			id, err := strconv.Atoi(quickstartId)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
				c.Abort()
				return
			}
			quickstart, err := FindQuickstartById(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"msg": err.Error()})
				c.Abort()
				return
			}

			c.Set("quickstart", &quickstart)
			c.Next()
		}
	}
}

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(subRouter *gin.RouterGroup) {
	subRouter.POST("", createQuickstart)
	subRouter.GET("", GetAllQuickstarts)
	entityRouter := subRouter.Group("/:id")
	entityRouter.Use(entityContext())
	entityRouter.GET("", getQuickstartById)
	entityRouter.DELETE("", deleteQuickstartById)
	entityRouter.PATCH("", updateQuickstartById)
}
