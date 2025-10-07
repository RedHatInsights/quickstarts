package services

import (
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/sirupsen/logrus"
)

// FavoriteService handles business logic for favorite quickstarts
type FavoriteService struct{}

// NewFavoriteService creates a new favorite service
func NewFavoriteService() *FavoriteService {
	return &FavoriteService{}
}

// GetFavorites gets all favorite quickstarts for a specific account
func (s *FavoriteService) GetFavorites(accountId string) ([]models.FavoriteQuickstart, error) {
	var favQuickstarts []models.FavoriteQuickstart
	result := database.DB.Where(&models.FavoriteQuickstart{AccountId: accountId, Favorite: true}).Find(&favQuickstarts)
	return favQuickstarts, result.Error
}

// SwitchFavorite toggles the favorite status for a quickstart
func (s *FavoriteService) SwitchFavorite(accountId string, quickstartName string, favorite bool) (models.FavoriteQuickstart, error) {
	var favQuickstart models.FavoriteQuickstart

	// First, find if the record exists
	findResult := database.DB.Where("account_id = ? AND quickstart_name = ?", accountId, quickstartName).First(&favQuickstart)

	if findResult.Error == nil {
		// Record exists, update it
		result := database.DB.Model(&favQuickstart).Update("favorite", favorite)
		if result.Error != nil {
			return favQuickstart, result.Error
		}
		return favQuickstart, nil
	}

	// Record doesn't exist, create a new one
	favQuickstart = models.FavoriteQuickstart{
		AccountId:      accountId,
		QuickstartName: quickstartName,
		Favorite:       favorite,
	}

	var qs models.Quickstart
	database.DB.Where("name = ?", quickstartName).Preload("FavoriteQuickstart").Find(&qs)
	qs.FavoriteQuickstart = append(qs.FavoriteQuickstart, favQuickstart)

	if err := database.DB.Save(&qs).Error; err != nil {
		logrus.Errorln("Error saving to database Quickstart:", err)
		return favQuickstart, err
	}

	return favQuickstart, nil
}
