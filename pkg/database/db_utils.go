package database

import (
	"fmt"
	"strings"
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

// execDB executes a GORM database operation with consistent logging and error handling
func (h *DBHelper) execDB(op, entityType string, id interface{}, fn func(*gorm.DB) *gorm.DB) error {
	h.logger.Debugf("%sing %s: %v", strings.Title(op), entityType, id)
	
	if err := fn(h.db).Error; err != nil {
		h.logger.Errorf("Failed to %s %s %v: %v", op, entityType, id, err)
		return &DBOperationError{
			Operation: op,
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully %sed %s: %v", op, entityType, id)
	return nil
}

// execAssoc executes an association operation with consistent logging and error handling
func (h *DBHelper) execAssoc(op, association, entityType string, id interface{}, fn func() error) error {
	h.logger.Debugf("%s %s associations for %s: %v", strings.Title(op), association, entityType, id)
	
	if err := fn(); err != nil {
		h.logger.Errorf("Failed to %s %s associations for %s %v: %v", op, association, entityType, id, err)
		return &DBOperationError{
			Operation: fmt.Sprintf("%s %s associations", op, association),
			Entity:    entityType,
			ID:        id,
			Err:       err,
		}
	}
	
	h.logger.Debugf("Successfully %sed %s associations for %s: %v", op, association, entityType, id)
	return nil
}

// Create performs a database create operation with consistent error handling
func (h *DBHelper) Create(entity interface{}, entityType string, id interface{}) error {
	return h.execDB("create", entityType, id, func(db *gorm.DB) *gorm.DB {
		return db.Create(entity)
	})
}

// Update performs a database update operation with consistent error handling
func (h *DBHelper) Update(entity interface{}, entityType string, id interface{}) error {
	return h.execDB("update", entityType, id, func(db *gorm.DB) *gorm.DB {
		return db.Save(entity)
	})
}

// Delete performs a database delete operation with consistent error handling
func (h *DBHelper) Delete(entity interface{}, entityType string, id interface{}) error {
	return h.execDB("delete", entityType, id, func(db *gorm.DB) *gorm.DB {
		return db.Unscoped().Delete(entity)
	})
}

// ClearAssociation clears an association with consistent error handling
func (h *DBHelper) ClearAssociation(entity interface{}, association string, entityType string, id interface{}) error {
	return h.execAssoc("clear", association, entityType, id, func() error {
		return h.db.Model(entity).Association(association).Clear()
	})
}

// AppendAssociation appends to an association with consistent error handling
func (h *DBHelper) AppendAssociation(entity interface{}, association string, values interface{}, entityType string, id interface{}) error {
	return h.execAssoc("add", association, entityType, id, func() error {
		return h.db.Model(entity).Association(association).Append(values)
	})
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
			errors = append(errors, fmt.Errorf("item %d: %w", i+1, err))
		} else {
			successCount++
		}
	}
	
	h.logger.Infof("Batch processing complete: %s - %d success, %d errors", batchName, successCount, len(errors))
	return successCount, errors
}

// ProcessBatchWithAggregatedError processes items and returns a single aggregated error if any fail
func (h *DBHelper) ProcessBatchWithAggregatedError(items []interface{}, processor func(interface{}) error, batchName string) (int, error) {
	successCount, errors := h.ProcessBatch(items, processor, batchName)
	
	if len(errors) == 0 {
		return successCount, nil
	}
	
	// Create aggregated error message
	errorMsgs := make([]string, len(errors))
	for i, err := range errors {
		errorMsgs[i] = err.Error()
	}
	
	aggregatedErr := fmt.Errorf("batch processing %s failed with %d errors: %s", 
		batchName, len(errors), strings.Join(errorMsgs, "; "))
	
	return successCount, aggregatedErr
}