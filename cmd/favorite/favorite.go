package main

import (
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/joho/godotenv"
)

func seedFavoriteQuickstart(quickstartName string) (models.FavoriteQuickstart, error) {

	var quickstart models.Quickstart
	r := database.DB.Where("name = ?", quickstartName).Preload("FavoriteQuickstart").Find(&quickstart)

	if r.RowsAffected != 0 {
		// Quickstart already exists
		fmt.Print(quickstartName, " is an existing quickstart.\n")
		// fmt.Print("Length of FavoriteQuickstarts inside quickstart ", quickstartName, " is ", len(quickstart.FavoriteQuickstart), "\n")
		return quickstart.FavoriteQuickstart[0], nil
	}

	// Create new quickstart
	mc := make(map[string]string)
	mc["foo"] = "bar"
	content, _ := json.Marshal(mc)

	quickstart = models.Quickstart{
		Name:               quickstartName,
		Content:            content,
		FavoriteQuickstart: []models.FavoriteQuickstart{},
	}

	favQuickstart := models.FavoriteQuickstart{
		AccountId:      "123",
		QuickstartName: quickstart.Name,
		Favorite:       true,
	}

	quickstart.FavoriteQuickstart = append(quickstart.FavoriteQuickstart, favQuickstart)
	database.DB.Create(&quickstart)
	fmt.Print(quickstartName, " quickstart has just been created.\n")
	fmt.Print("Favorite was set to '", favQuickstart.Favorite, "'\n")
	// fmt.Print("Length of FavoriteQuickstarts inside quickstart ", quickstartName, " is ", len(quickstart.FavoriteQuickstart), "\n")

	return favQuickstart, nil
}

func main() {
	godotenv.Load()
	config.Init()
	database.Init()

	// var favQuickstart models.FavoriteQuickstart
	var err error

	_, err = seedFavoriteQuickstart("test-test-test")
	if err != nil {
		fmt.Println("Unable to seed favoriteQuickstart: ", err.Error())
	}
}
