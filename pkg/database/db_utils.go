package database

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

// execOp executes any operation with consistent logging and error handling
func (h *DBHelper) execOp(op, entityType string, id interface{}, actionCallback func() error) error {
	h.logger.Debugf("%s %s: %v", strings.Title(op), entityType, id)

	if err := actionCallback(); err != nil {
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

// Create performs a database create operation with consistent error handling
func (h *DBHelper) Create(entity interface{}, entityType string, id interface{}) error {
	return h.execOp("create", entityType, id, func() error {
		return h.db.Create(entity).Error
	})
}

// Update performs a database update operation with consistent error handling
func (h *DBHelper) Update(entity interface{}, entityType string, id interface{}) error {
	return h.execOp("update", entityType, id, func() error {
		return h.db.Save(entity).Error
	})
}

// Delete performs a database delete operation with consistent error handling
func (h *DBHelper) Delete(entity interface{}, entityType string, id interface{}) error {
	return h.execOp("delete", entityType, id, func() error {
		return h.db.Unscoped().Delete(entity).Error
	})
}

// ClearAssociation clears an association with consistent error handling
func (h *DBHelper) ClearAssociation(entity interface{}, association string, entityType string, id interface{}) error {
	op := fmt.Sprintf("clear %s associations", association)
	return h.execOp(op, entityType, id, func() error {
		return h.db.Model(entity).Association(association).Clear()
	})
}

// AppendAssociation appends an association with consistent error handling
func (h *DBHelper) AppendAssociation(entity interface{}, association string, values interface{}, entityType string, id interface{}) error {
	op := fmt.Sprintf("add %s associations", association)
	return h.execOp(op, entityType, id, func() error {
		return h.db.Model(entity).Association(association).Append(values)
	})
}

// FindFirst finds the first instance of an entity matching the WHERE condition
func (h *DBHelper) FindFirst(entity interface{}, where interface{}, entityType string, id interface{}) (result *gorm.DB, errorInfo error) {
	h.logger.Debugf("Finding %s: %v", entityType, id)

	result = h.db.Where(where).First(entity)
	if result.Error != nil {
		h.logger.Errorf("Failed to find %s %v: %v", entityType, id, result.Error)
		return result, &DBOperationError{
			Operation: "find",
			Entity:    entityType,
			ID:        id,
			Err:       result.Error,
		}
	}

	h.logger.Debugf("Found %s: %v", entityType, id)
	return result, nil
}

// FindAll finds all instances of an entity matching the WHERE condition
func (h *DBHelper) FindAll(entities interface{}, where interface{}, entityType string) error {
	h.logger.Debugf("Finding all %s", entityType)

	var result *gorm.DB
	if where != nil {
		result = h.db.Where(where).Find(entities)
	} else {
		result = h.db.Find(entities)
	}

	if result.Error != nil {
		h.logger.Errorf("Failed to find all %s: %v", entityType, result.Error)
		return &DBOperationError{
			Operation: "find all",
			Entity:    entityType,
			ID:        nil,
			Err:       result.Error,
		}
	}

	h.logger.Debugf("Found %d %s records", result.RowsAffected, entityType)
	return nil
}

// FindOrCreate finds an existing record or creates a new one
func (h *DBHelper) FindOrCreate(entity interface{}, where interface{}, entityType string, id interface{}) (wasCreated bool, errorInfo error) {
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

	// Note: When no record is created, RowsAffected is 0
	created := result.RowsAffected > 0
	if created {
		h.logger.Debugf("Created new %s: %v", entityType, id)
	} else {
		h.logger.Debugf("Found existing %s: %v", entityType, id)
	}

	return created, nil
}

// ProcessBatch processes a batch of items with error collection and progress logging
// Returns (successCount, errors) - callers can aggregate errors if needed
func (h *DBHelper) ProcessBatch(items []interface{}, processorCallback func(interface{}) error, batchName string) (successCount int, errors []error) {
	h.logger.Infof("Starting batch %s (%d items)", batchName, len(items))

	for i, item := range items {
		if err := processorCallback(item); err != nil {
			errors = append(errors, fmt.Errorf("item %d: %w", i+1, err))
		} else {
			successCount++
		}
	}

	h.logger.Infof("Finished batch %s: %d success, %d errors", batchName, successCount, len(errors))
	return
}
