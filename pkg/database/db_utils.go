package database

import (
	"fmt"
	"gorm.io/gorm"
	"github.com/sirupsen/logrus"
)

// DBOperationError wraps database operations with consistent error handling and logging
type DBOperationError struct {
	Operation string
	Entity    string
	ID        interface{}
	Err       error
}

func (e *DBOperationError) Error() string {
	if e.ID != nil {
		return fmt.Sprintf("failed to %s %s %v: %v", e.Operation, e.Entity, e.ID, e.Err)
	}
	return fmt.Sprintf("failed to %s %s: %v", e.Operation, e.Entity, e.Err)
}

func (e *DBOperationError) Unwrap() error {
	return e.Err
}

// DBHelper provides consistent database operations with error handling and logging
type DBHelper struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewDBHelper creates a new database helper with consistent logging
func NewDBHelper(db *gorm.DB, context string) *DBHelper {
	return &DBHelper{
		db:     db,
		logger: logrus.WithField("context", context),
	}
}

// Create performs a database create operation with consistent error handling
func (h *DBHelper) Create(entity interface{}, entityType string, id interface{}) error {
	h.logger.Debugf("Creating %s: %v", entityType, id)
	
	if err := h.db.Create(entity).Error; err != nil {
		h.logger.Errorf("Failed to create %s %v: %v", entityType, id, err)
		return &DBOperationError{
			Operation: "create",
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully created %s: %v", entityType, id)
	return nil
}

// Update performs a database update operation with consistent error handling
func (h *DBHelper) Update(entity interface{}, entityType string, id interface{}) error {
	h.logger.Debugf("Updating %s: %v", entityType, id)
	
	if err := h.db.Save(entity).Error; err != nil {
		h.logger.Errorf("Failed to update %s %v: %v", entityType, id, err)
		return &DBOperationError{
			Operation: "update",
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully updated %s: %v", entityType, id)
	return nil
}

// Delete performs a database delete operation with consistent error handling
func (h *DBHelper) Delete(entity interface{}, entityType string, id interface{}) error {
	h.logger.Debugf("Deleting %s: %v", entityType, id)
	
	if err := h.db.Unscoped().Delete(entity).Error; err != nil {
		h.logger.Errorf("Failed to delete %s %v: %v", entityType, id, err)
		return &DBOperationError{
			Operation: "delete",
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully deleted %s: %v", entityType, id)
	return nil
}

// ClearAssociation clears an association with consistent error handling
func (h *DBHelper) ClearAssociation(entity interface{}, association string, entityType string, id interface{}) error {
	h.logger.Debugf("Clearing %s associations for %s: %v", association, entityType, id)
	
	if err := h.db.Model(entity).Association(association).Clear(); err != nil {
		h.logger.Errorf("Failed to clear %s associations for %s %v: %v", association, entityType, id, err)
		return &DBOperationError{
			Operation: fmt.Sprintf("clear %s associations", association),
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully cleared %s associations for %s: %v", association, entityType, id)
	return nil
}

// AppendAssociation appends to an association with consistent error handling
func (h *DBHelper) AppendAssociation(entity interface{}, association string, values interface{}, entityType string, id interface{}) error {
	h.logger.Debugf("Adding %s association for %s: %v", association, entityType, id)
	
	if err := h.db.Model(entity).Association(association).Append(values); err != nil {
		h.logger.Errorf("Failed to add %s association for %s %v: %v", association, entityType, id, err)
		return &DBOperationError{
			Operation: fmt.Sprintf("add %s association", association),
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully added %s association for %s: %v", association, entityType, id)
	return nil
}

// FindOrCreate finds an existing record or creates a new one
func (h *DBHelper) FindOrCreate(entity interface{}, where interface{}, entityType string, id interface{}) (bool, error) {
	h.logger.Debugf("Finding or creating %s: %v", entityType, id)
	
	result := h.db.Where(where).FirstOrCreate(entity)
	if result.Error != nil {
		h.logger.Errorf("Failed to find or create %s %v: %v", entityType, id, result.Error)
		return false, &DBOperationError{
			Operation: "find or create",
			Entity:    entityType,
			ID:        id,
			Err:       result.Error,
		}
	}
	
	created := result.RowsAffected > 0
	if created {
		h.logger.Debugf("Created new %s: %v", entityType, id)
	} else {
		h.logger.Debugf("Found existing %s: %v", entityType, id)
	}
	
	return created, nil
}

// ProcessBatch processes a batch of items with error collection and progress logging
func (h *DBHelper) ProcessBatch(items []interface{}, processor func(interface{}) error, batchName string) (int, []error) {
	h.logger.Infof("Starting batch processing: %s (%d items)", batchName, len(items))
	
	var errors []error
	successCount := 0
	
	for i, item := range items {
		h.logger.Debugf("Processing item %d/%d in batch %s", i+1, len(items), batchName)
		
		if err := processor(item); err != nil {
			h.logger.Errorf("Error processing item %d in batch %s: %v", i+1, batchName, err)
			errors = append(errors, err)
		} else {
			successCount++
		}
	}
	
	h.logger.Infof("Batch processing complete: %s - %d success, %d errors", batchName, successCount, len(errors))
	return successCount, errors
}