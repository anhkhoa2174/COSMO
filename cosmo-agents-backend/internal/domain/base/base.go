package base

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base contains common columns for all tables
// Equivalent to Python's core/db/Base
type Base struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
}

// TimestampMixin adds created_at and updated_at fields
// Equivalent to Python's core/db/mixins/TimestampMixin
type TimestampMixin struct {
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// SoftDeleteMixin adds soft delete functionality
// Equivalent to Python's core/db/mixins/SoftDeleteMixin
type SoftDeleteMixin struct {
	IsDeleted bool `gorm:"default:false;index" json:"is_deleted"`
}

// BeforeCreate hook to set UUID before creating record
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// JSON is a custom type for JSON fields
type JSON []byte

// Scan implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = v
	case string:
		*j = []byte(v)
	default:
		return json.Unmarshal([]byte{}, j)
	}
	return nil
}

// Value implements driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// Unmarshal unmarshals JSON data into the provided value
func (j JSON) Unmarshal(v interface{}) error {
	if j == nil {
		return nil
	}
	return json.Unmarshal([]byte(j), v)
}

// Marshal creates JSON from the provided value
func (j *JSON) Marshal(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*j = JSON(data)
	return nil
}

// JSONB is a custom type for JSONB fields (PostgreSQL)
type JSONB []byte

// Scan implements sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("{}")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*j = JSONB("{}")
		} else {
			*j = v
		}
	case string:
		if v == "" {
			*j = JSONB("{}")
		} else {
			*j = JSONB(v)
		}
	default:
		return json.Unmarshal([]byte("{}"), j)
	}
	return nil
}

// Value implements driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return []byte(j), nil
}

// Unmarshal unmarshals JSONB data into the provided value
func (j JSONB) Unmarshal(v interface{}) error {
	if len(j) == 0 {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal([]byte(j), v)
}

// Marshal creates JSONB from the provided value
func (j *JSONB) Marshal(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*j = JSONB(data)
	return nil
}
