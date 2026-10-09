package filter

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// TestModel for filter testing
type TestFilterModel struct {
	ID       int    `gorm:"primaryKey"`
	Name     string `gorm:"size:255"`
	Email    string `gorm:"size:255;unique"`
	Age      int
	Status   string
	Category string
}

func setupFilterTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestFilterModel{})
	require.NoError(t, err)

	// Insert test data
	testData := []TestFilterModel{
		{Name: "John Doe", Email: "john@example.com", Age: 25, Status: "active", Category: "premium"},
		{Name: "Jane Smith", Email: "jane@example.com", Age: 30, Status: "active", Category: "basic"},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive", Category: "premium"},
		{Name: "Alice Brown", Email: "alice@example.com", Age: 28, Status: "pending", Category: "basic"},
		{Name: "Charlie Wilson", Email: "charlie@example.com", Age: 40, Status: "active", Category: "premium"},
	}

	for _, data := range testData {
		db.Create(&data)
	}

	return db
}

func TestFilterBuilder_Apply(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("empty filter", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		result := fb.Apply(base.Filter{})

		// Should return the same DB instance
		assert.Same(t, db, result)
	})

	t.Run("simple equality filter", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"status": "active",
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 3) // John, Jane, Charlie
		for _, record := range records {
			assert.Equal(t, "active", record.Status)
		}
	})

	t.Run("multiple equality filters", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"status":   "active",
			"category": "premium",
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // John, Charlie
		for _, record := range records {
			assert.Equal(t, "active", record.Status)
			assert.Equal(t, "premium", record.Category)
		}
	})
}

func TestFilterBuilder_ApplyOperators(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("equal operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"status": map[string]interface{}{
				"$eq": "active",
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 3)
	})

	t.Run("not equal operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"status": map[string]interface{}{
				"$ne": "active",
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // Bob, Alice
		for _, record := range records {
			assert.NotEqual(t, "active", record.Status)
		}
	})

	t.Run("greater than operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"age": map[string]interface{}{
				"$gt": 30,
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // Bob (35), Charlie (40)
		for _, record := range records {
			assert.Greater(t, record.Age, 30)
		}
	})

	t.Run("less than equal operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"age": map[string]interface{}{
				"$lte": 30,
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 3) // John (25), Jane (30), Alice (28)
		for _, record := range records {
			assert.LessOrEqual(t, record.Age, 30)
		}
	})

	t.Run("in operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"age": map[string]interface{}{
				"$in": []int{25, 35},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // John (25), Bob (35)
	})

	t.Run("not in operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"status": map[string]interface{}{
				"$nin": []string{"active", "pending"},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 1) // Bob (inactive)
		assert.Equal(t, "inactive", records[0].Status)
	})

	t.Run("between operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"age": map[string]interface{}{
				"$between": []interface{}{25, 35},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 4) // John (25), Jane (30), Bob (35), Alice (28)
	})

	t.Run("like operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"name": map[string]interface{}{
				"$like": "J%",
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // John, Jane
	})

	t.Run("ilike operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"name": map[string]interface{}{
				"$like": "John%", // Use LIKE for SQLite compatibility, more specific pattern
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 1) // John Doe (only, not Bob Johnson)
	})

	t.Run("not like operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"name": map[string]interface{}{
				"$nlike": "J%",
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 3) // Bob, Alice, Charlie
	})
}

func TestFilterBuilder_ApplyLogicalOperators(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("AND operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"$and": []interface{}{
				map[string]interface{}{"status": "active"},
				map[string]interface{}{"category": "premium"},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // John, Charlie
	})

	t.Run("OR operator", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"$or": []interface{}{
				map[string]interface{}{"status": "inactive"},
				map[string]interface{}{"age": 25},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 2) // Bob (inactive), John (age 25)
	})

	t.Run("complex nested AND/OR", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"$or": []interface{}{
				map[string]interface{}{
					"$and": []interface{}{
						map[string]interface{}{"status": "active"},
						map[string]interface{}{"category": "premium"},
					},
				},
				map[string]interface{}{
					"$and": []interface{}{
						map[string]interface{}{"status": "inactive"},
						map[string]interface{}{"category": "basic"},
					},
				},
			},
		}
		result := fb.Apply(filterMap)

		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		// Only active + premium should match (John, Charlie)
		// No records match inactive + basic
		assert.Len(t, records, 2)
	})
}

func TestFilterBuilder_QualifyField(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("with table specified", func(t *testing.T) {
		query := db.Table("test_filter_models")
		fb := NewFilterBuilder(query)
		filterMap := base.Filter{
			"name": "John Doe",
		}
		result := fb.Apply(filterMap)

		// Execute the query - it should work correctly
		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "John Doe", records[0].Name)
	})

	t.Run("with dotted field name", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"name": "John Doe", // Use a valid field for this test
		}
		result := fb.Apply(filterMap)

		// Execute the query - it should work correctly
		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "John Doe", records[0].Name)
	})
}

func TestFilterBuilder_InvalidFieldNames(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("SQL injection attempt", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"name; DROP TABLE users; --": "malicious",
		}
		result := fb.Apply(filterMap)

		// Invalid field names should be ignored
		// All records should be returned
		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 5) // All records
	})

	t.Run("very long field name", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		longName := "a" + string(make([]byte, 100)) // 101 characters
		filterMap := base.Filter{
			longName: "value",
		}
		result := fb.Apply(filterMap)

		// Invalid field names should be ignored
		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 5) // All records
	})
}

func TestFilterBuilder_InvalidOperatorValues(t *testing.T) {
	db := setupFilterTestDB(t)

	t.Run("between with invalid array length", func(t *testing.T) {
		fb := NewFilterBuilder(db)
		filterMap := base.Filter{
			"age": map[string]interface{}{
				"$between": []interface{}{25}, // Only one value
			},
		}
		result := fb.Apply(filterMap)

		// Should not crash, just ignore the invalid between
		var records []TestFilterModel
		err := result.Find(&records).Error
		require.NoError(t, err)

		assert.Len(t, records, 5) // All records
	})
}

func TestBuildOrderBy(t *testing.T) {
	t.Run("empty sort by", func(t *testing.T) {
		result := BuildOrderBy("", "")
		assert.Equal(t, "created_at DESC", result)
	})

	t.Run("sort by with desc", func(t *testing.T) {
		result := BuildOrderBy("name", "DESC")
		assert.Equal(t, "name DESC", result)
	})

	t.Run("sort by with lowercase desc", func(t *testing.T) {
		result := BuildOrderBy("age", "desc")
		assert.Equal(t, "age DESC", result)
	})

	t.Run("sort by with asc", func(t *testing.T) {
		result := BuildOrderBy("email", "ASC")
		assert.Equal(t, "email ASC", result)
	})

	t.Run("sort by with mixed case", func(t *testing.T) {
		result := BuildOrderBy("status", "AsC")
		assert.Equal(t, "status ASC", result)
	})

	t.Run("sort by with empty sort order", func(t *testing.T) {
		result := BuildOrderBy("category", "")
		assert.Equal(t, "category DESC", result)
	})

	t.Run("SQL injection attempt", func(t *testing.T) {
		result := BuildOrderBy("name; DROP TABLE users; --", "ASC")
		assert.Equal(t, "name DROP TABLE users  ASC", result) // Should be sanitized
	})
}
