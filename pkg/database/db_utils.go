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

// execOp executes any operation with consistent logging and error handling
func (h *DBHelper) execOp(op, entityType string, id interface{}, action func() error) error {
	h.logger.Debugf("%s %s: %v", strings.Title(op), entityType, id)
	
	if err := action(); err != nil {
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

// AppendAssociation appends to an association with consistent error handling
func (h *DBHelper) AppendAssociation(entity interface{}, association string, values interface{}, entityType string, id interface{}) error {
	op := fmt.Sprintf("add %s associations", association)
	return h.execOp(op, entityType, id, func() error {
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
// Returns (successCount, errors) - callers can aggregate errors if needed
func (h *DBHelper) ProcessBatch(items []interface{}, processor func(interface{}) error, batchName string) (successCount int, errors []error) {
	h.logger.Infof("Starting batch %s (%d items)", batchName, len(items))
	
	for i, item := range items {
		if err := processor(item); err != nil {
			errors = append(errors, fmt.Errorf("item %d: %w", i+1, err))
		} else {
			successCount++
		}
	}
	
	h.logger.Infof("Finished batch %s: %d success, %d errors", batchName, successCount, len(errors))
	return
}