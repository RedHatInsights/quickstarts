package database

import (
	"fmt"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"gorm.io/gorm"
)

var db *gorm.DB

// CreateQuickstart is a function that creates new quickstart entry in the DB
func CreateQuickstart(quickstart *models.Quickstart) (uint, error) {
	newQuickstart := models.Quickstart{Title: "quickstart.Title"}
	fmt.Println(newQuickstart)
	result := db.Create(&newQuickstart)
	fmt.Println(result, newQuickstart.ID)
	return newQuickstart.ID, nil
}

// GetQuickstarts list all avaiable quickstarts
func GetQuickstarts() ([]models.Quickstart, error) {
	var quickStarts []models.Quickstart
	db.Find(quickStarts)
	return quickStarts, nil
}
