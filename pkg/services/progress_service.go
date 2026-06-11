package services

import (
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"gorm.io/datatypes"
)

// ProgressService handles business logic for quickstart progress
type ProgressService struct{}

// NewProgressService creates a new progress service
func NewProgressService() *ProgressService {
	return &ProgressService{}
}

// GetExistingProgress finds existing progress record by name and account ID
func (s *ProgressService) GetExistingProgress(name string, accountId int) (models.QuickstartProgress, error) {
	var progress models.QuickstartProgress
	var where models.QuickstartProgress
	where.QuickstartName = name
	where.AccountId = accountId
	err := database.DB.Where(where).First(&progress).Error
	return progress, err
}

// GetAllProgress returns all progress records
func (s *ProgressService) GetAllProgress() ([]models.QuickstartProgress, error) {
	var progress []models.QuickstartProgress
	err := database.DB.Find(&progress).Error
	return progress, err
}

// GetProgress returns progress records with optional filtering by account and/or quickstart name
func (s *ProgressService) GetProgress(accountId *int, quickstartName *string) ([]models.QuickstartProgress, error) {
	var progresses []models.QuickstartProgress
	var where models.QuickstartProgress

	if accountId != nil {
		where.AccountId = *accountId
	}

	if quickstartName != nil {
		where.QuickstartName = *quickstartName
	}

	err := database.DB.Where(where).Find(&progresses).Error
	return progresses, err
}

// UpdateProgress creates new progress or updates existing progress
func (s *ProgressService) UpdateProgress(accountId int, quickstartName string, progress *datatypes.JSON) (models.QuickstartProgress, error) {
	currentProgress, err := s.GetExistingProgress(quickstartName, accountId)

	// If no progress exists for this name and account, create new
	if err != nil {
		newProgress := models.QuickstartProgress{
			AccountId:      accountId,
			QuickstartName: quickstartName,
			Progress:       progress,
		}
		err = database.DB.Create(&newProgress).Error
		return newProgress, err
	}

	// Update existing progress
	currentProgress.Progress = progress
	err = database.DB.Save(&currentProgress).Error
	return currentProgress, err
}

// DeleteProgress deletes progress record by ID
func (s *ProgressService) DeleteProgress(id int) error {
	var quickStartProgress models.QuickstartProgress

	// First check if record exists
	err := database.DB.First(&quickStartProgress, id).Error
	if err != nil {
		return err
	}

	// Delete the record
	err = database.DB.Delete(&quickStartProgress).Error
	return err
}
