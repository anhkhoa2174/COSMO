package base

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModel represents a model that uses all base mixins
type TestModel struct {
	Base
	TimestampMixin
	SoftDeleteMixin
	Name string `gorm:"column:name" json:"name"`
	Data JSON   `gorm:"column:data" json:"data"`
	Meta JSONB  `gorm:"column:meta" json:"meta"`
}

// TableName specifies the table name for TestModel
func (TestModel) TableName() string {
	return "test_models"
}

// TestBase_IDGeneration tests the ID generation in Base struct
func TestBase_IDGeneration(t *testing.T) {
	t.Run("New UUID created when ID is nil", func(t *testing.T) {
		model := &TestModel{}

		// Simulate BeforeCreate with nil transaction (for this test)
		err := model.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, model.ID, "ID should be set to a non-nil UUID")
	})

	t.Run("Existing UUID is preserved", func(t *testing.T) {
		existingID := uuid.New()
		model := &TestModel{}
		model.ID = existingID

		err := model.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, existingID, model.ID, "Existing ID should be preserved")
	})

	t.Run("JSON serialization includes ID", func(t *testing.T) {
		testID := uuid.New()
		model := &TestModel{}
		model.ID = testID
		model.Name = "Test"

		data, err := json.Marshal(model)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, testID.String(), result["id"], "JSON should contain the UUID")
	})
}

// TestBase_UnmarshalJSON tests JSON unmarshaling for Base struct
func TestBase_UnmarshalJSON(t *testing.T) {
	t.Run("Valid UUID from JSON", func(t *testing.T) {
		testID := uuid.New()
		jsonData := `{"id": "` + testID.String() + `", "name": "Test"}`

		var model TestModel
		err := json.Unmarshal([]byte(jsonData), &model)

		assert.NoError(t, err)
		assert.Equal(t, testID, model.ID)
		assert.Equal(t, "Test", model.Name)
	})

	t.Run("Invalid UUID from JSON", func(t *testing.T) {
		jsonData := `{"id": "invalid-uuid", "name": "Test"}`

		var model TestModel
		err := json.Unmarshal([]byte(jsonData), &model)

		assert.Error(t, err)
	})

	t.Run("Missing ID in JSON", func(t *testing.T) {
		jsonData := `{"name": "Test"}`

		var model TestModel
		err := json.Unmarshal([]byte(jsonData), &model)

		assert.NoError(t, err)
		assert.Equal(t, uuid.Nil, model.ID)
	})
}

// TestTimestampMixin tests the TimestampMixin functionality
func TestTimestampMixin(t *testing.T) {
	t.Run("Zero values initially", func(t *testing.T) {
		model := &TestModel{}

		assert.True(t, model.CreatedAt.IsZero(), "CreatedAt should be zero initially")
		assert.True(t, model.UpdatedAt.IsZero(), "UpdatedAt should be zero initially")
	})

	t.Run("JSON serialization includes timestamps", func(t *testing.T) {
		now := time.Now().UTC()
		model := &TestModel{
			TimestampMixin: TimestampMixin{
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "Test",
		}

		data, err := json.Marshal(model)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Contains(t, result, "created_at")
		assert.Contains(t, result, "updated_at")
	})

	t.Run("JSON unmarshaling timestamps", func(t *testing.T) {
		now := time.Now().UTC()
		jsonData := `{
			"created_at": "` + now.Format(time.RFC3339Nano) + `",
			"updated_at": "` + now.Add(time.Hour).Format(time.RFC3339Nano) + `",
			"name": "Test"
		}`

		var model TestModel
		err := json.Unmarshal([]byte(jsonData), &model)

		assert.NoError(t, err)
		assert.False(t, model.CreatedAt.IsZero())
		assert.False(t, model.UpdatedAt.IsZero())
	})

	t.Run("Timestamp comparison", func(t *testing.T) {
		createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)

		model := &TestModel{
			TimestampMixin: TimestampMixin{
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		}

		assert.True(t, model.UpdatedAt.After(model.CreatedAt))
	})
}

// TestSoftDeleteMixin tests the SoftDeleteMixin functionality
func TestSoftDeleteMixin(t *testing.T) {
	t.Run("Default is not deleted", func(t *testing.T) {
		model := &TestModel{}

		assert.False(t, model.IsDeleted, "IsDeleted should default to false")
	})

	t.Run("JSON serialization includes is_deleted", func(t *testing.T) {
		model := &TestModel{
			SoftDeleteMixin: SoftDeleteMixin{IsDeleted: true},
			Name:            "Deleted Model",
		}

		data, err := json.Marshal(model)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, true, result["is_deleted"])
	})

	t.Run("JSON unmarshaling is_deleted", func(t *testing.T) {
		jsonData := `{"is_deleted": true, "name": "Test"}`

		var model TestModel
		err := json.Unmarshal([]byte(jsonData), &model)

		assert.NoError(t, err)
		assert.True(t, model.IsDeleted)
	})
}

// TestJSON tests the JSON custom type
func TestJSON(t *testing.T) {
	t.Run("Nil JSON Scan", func(t *testing.T) {
		var j JSON
		err := j.Scan(nil)

		assert.NoError(t, err)
		assert.Nil(t, j, "JSON should be nil after scanning nil")
	})

	t.Run("Scan from bytes", func(t *testing.T) {
		var j JSON
		testBytes := []byte(`{"key": "value"}`)

		err := j.Scan(testBytes)

		assert.NoError(t, err)
		assert.Equal(t, testBytes, []byte(j))
	})

	t.Run("Scan from string", func(t *testing.T) {
		var j JSON
		testString := `{"key": "value"}`

		err := j.Scan(testString)

		assert.NoError(t, err)
		assert.Equal(t, []byte(testString), []byte(j))
	})

	t.Run("Scan from unsupported type", func(t *testing.T) {
		var j JSON
		testValue := 123

		err := j.Scan(testValue)

		assert.Error(t, err)
	})

	t.Run("Value with nil JSON", func(t *testing.T) {
		var j JSON
		value, err := j.Value()

		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Value with JSON data", func(t *testing.T) {
		j := JSON(`{"key": "value"}`)
		value, err := j.Value()

		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"key": "value"}`), value)
	})

	t.Run("Marshal data to JSON", func(t *testing.T) {
		var j JSON
		testData := map[string]interface{}{
			"name":  "Test",
			"value": 42,
		}

		err := j.Marshal(testData)

		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(j), &result)
		require.NoError(t, err)

		assert.Equal(t, "Test", result["name"])
		assert.Equal(t, float64(42), result["value"])
	})

	t.Run("Unmarshal data from JSON", func(t *testing.T) {
		j := JSON(`{"name": "Test", "value": 42}`)

		var result struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		}

		err := j.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, float64(42), result.Value)
	})

	t.Run("Unmarshal from nil JSON", func(t *testing.T) {
		var j JSON

		var result map[string]interface{}
		err := j.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})

	t.Run("Marshal with complex data", func(t *testing.T) {
		var j JSON
		testData := struct {
			ID      int                    `json:"id"`
			Name    string                 `json:"name"`
			Tags    []string               `json:"tags"`
			Details map[string]interface{} `json:"details"`
		}{
			ID:   1,
			Name: "Complex Object",
			Tags: []string{"tag1", "tag2", "tag3"},
			Details: map[string]interface{}{
				"active": true,
				"score":  95.5,
			},
		}

		err := j.Marshal(testData)
		require.NoError(t, err)

		var unmarshaled struct {
			ID      int                    `json:"id"`
			Name    string                 `json:"name"`
			Tags    []string               `json:"tags"`
			Details map[string]interface{} `json:"details"`
		}

		err = j.Unmarshal(&unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, testData, unmarshaled)
	})

	t.Run("Driver.Valuer interface implementation", func(t *testing.T) {
		var j driver.Valuer = JSON(`{"test": "value"}`)

		value, err := j.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)
	})
}

// TestJSONB tests the JSONB custom type
func TestJSONB(t *testing.T) {
	t.Run("Nil JSONB Scan", func(t *testing.T) {
		var j JSONB
		err := j.Scan(nil)

		assert.NoError(t, err)
		assert.Equal(t, JSONB("{}"), j, "JSONB should default to empty object")
	})

	t.Run("Scan from empty bytes", func(t *testing.T) {
		var j JSONB
		err := j.Scan([]byte{})

		assert.NoError(t, err)
		assert.Equal(t, JSONB("{}"), j, "JSONB should default to empty object for empty bytes")
	})

	t.Run("Scan from empty string", func(t *testing.T) {
		var j JSONB
		err := j.Scan("")

		assert.NoError(t, err)
		assert.Equal(t, JSONB("{}"), j, "JSONB should default to empty object for empty string")
	})

	t.Run("Scan from valid bytes", func(t *testing.T) {
		var j JSONB
		testBytes := []byte(`{"key": "value"}`)

		err := j.Scan(testBytes)

		assert.NoError(t, err)
		assert.Equal(t, testBytes, []byte(j))
	})

	t.Run("Scan from valid string", func(t *testing.T) {
		var j JSONB
		testString := `{"key": "value"}`

		err := j.Scan(testString)

		assert.NoError(t, err)
		assert.Equal(t, JSONB(testString), j)
	})

	t.Run("Scan from unsupported type", func(t *testing.T) {
		var j JSONB
		testValue := 123

		err := j.Scan(testValue)

		assert.Error(t, err)
	})

	t.Run("Value with empty JSONB", func(t *testing.T) {
		j := JSONB("")
		value, err := j.Value()

		assert.NoError(t, err)
		assert.Equal(t, "{}", value)
	})

	t.Run("Value with JSONB data", func(t *testing.T) {
		j := JSONB(`{"key": "value"}`)
		value, err := j.Value()

		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"key": "value"}`), value)
	})

	t.Run("Unmarshal from empty JSONB", func(t *testing.T) {
		j := JSONB("")

		var result map[string]interface{}
		err := j.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})

	t.Run("Unmarshal from nil JSONB", func(t *testing.T) {
		var j JSONB
		j = nil

		var result map[string]interface{}
		err := j.Unmarshal(&result)

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})

	t.Run("Marshal to JSONB", func(t *testing.T) {
		var j JSONB
		testData := map[string]interface{}{
			"name":  "Test",
			"value": 42,
		}

		err := j.Marshal(testData)

		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(j), &result)
		require.NoError(t, err)

		assert.Equal(t, "Test", result["name"])
		assert.Equal(t, float64(42), result["value"])
	})

	t.Run("JSONB with nested objects", func(t *testing.T) {
		var j JSONB
		testData := map[string]interface{}{
			"user": map[string]interface{}{
				"id":   1,
				"name": "John Doe",
			},
			"settings": map[string]interface{}{
				"theme":    "dark",
				"language": "en",
			},
		}

		err := j.Marshal(testData)
		require.NoError(t, err)

		var unmarshaled map[string]interface{}
		err = j.Unmarshal(&unmarshaled)
		assert.NoError(t, err)

		user := unmarshaled["user"].(map[string]interface{})
		assert.Equal(t, float64(1), user["id"])
		assert.Equal(t, "John Doe", user["name"])

		settings := unmarshaled["settings"].(map[string]interface{})
		assert.Equal(t, "dark", settings["theme"])
		assert.Equal(t, "en", settings["language"])
	})

	t.Run("JSONB with arrays", func(t *testing.T) {
		var j JSONB
		testData := []string{"item1", "item2", "item3"}

		err := j.Marshal(testData)
		require.NoError(t, err)

		var unmarshaled []string
		err = j.Unmarshal(&unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, testData, unmarshaled)
	})
}

// TestModelIntegration tests integration of all mixins
func TestModelIntegration(t *testing.T) {
	t.Run("Complete model serialization", func(t *testing.T) {
		testID := uuid.New()
		now := time.Now().UTC()
		testJSONData := map[string]interface{}{
			"field1": "value1",
			"field2": 42,
		}
		testJSONBData := map[string]interface{}{
			"meta1": true,
			"meta2": "test",
		}

		// Set up JSON fields
		var jsonData JSON
		err := jsonData.Marshal(testJSONData)
		require.NoError(t, err)

		var jsonbData JSONB
		err = jsonbData.Marshal(testJSONBData)
		require.NoError(t, err)

		model := &TestModel{
			Base: Base{ID: testID},
			TimestampMixin: TimestampMixin{
				CreatedAt: now,
				UpdatedAt: now.Add(time.Hour),
			},
			SoftDeleteMixin: SoftDeleteMixin{IsDeleted: false},
			Name:            "Integration Test",
			Data:            jsonData,
			Meta:            jsonbData,
		}

		// Serialize to JSON
		serialized, err := json.Marshal(model)
		require.NoError(t, err)

		// Deserialize back
		var deserialized TestModel
		err = json.Unmarshal(serialized, &deserialized)
		require.NoError(t, err)

		// Verify all fields
		assert.Equal(t, testID, deserialized.ID)
		assert.Equal(t, now.Unix(), deserialized.CreatedAt.Unix())
		assert.Equal(t, now.Add(time.Hour).Unix(), deserialized.UpdatedAt.Unix())
		assert.False(t, deserialized.IsDeleted)
		assert.Equal(t, "Integration Test", deserialized.Name)

		// Verify JSON data
		var unmarshaledJSON map[string]interface{}
		err = deserialized.Data.Unmarshal(&unmarshaledJSON)
		assert.NoError(t, err)
		assert.Equal(t, "value1", unmarshaledJSON["field1"])

		// Verify JSONB data
		var unmarshaledJSONB map[string]interface{}
		err = deserialized.Meta.Unmarshal(&unmarshaledJSONB)
		assert.NoError(t, err)
		assert.Equal(t, true, unmarshaledJSONB["meta1"])
	})

	t.Run("Empty model defaults", func(t *testing.T) {
		model := &TestModel{}

		// BeforeCreate should set ID
		err := model.BeforeCreate(nil)
		assert.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, model.ID)
		assert.True(t, model.CreatedAt.IsZero())
		assert.True(t, model.UpdatedAt.IsZero())
		assert.False(t, model.IsDeleted)
		assert.Empty(t, model.Name)
		assert.Nil(t, model.Data)
		// JSONB is nil until scanned from database or explicitly set
		assert.Nil(t, model.Meta)
	})
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("JSON with invalid JSON data", func(t *testing.T) {
		j := JSON(`{invalid json}`)

		var result map[string]interface{}
		err := j.Unmarshal(&result)
		assert.Error(t, err)
	})

	t.Run("JSON Marshal error", func(t *testing.T) {
		var j JSON

		// Try to marshal a function (which is not JSON serializable)
		err := j.Marshal(func() {})
		assert.Error(t, err)
	})

	t.Run("JSONB with invalid JSON data", func(t *testing.T) {
		j := JSONB(`{invalid json}`)

		var result map[string]interface{}
		err := j.Unmarshal(&result)
		assert.Error(t, err)
	})

	t.Run("Large JSON data", func(t *testing.T) {
		var j JSON

		// Create a large JSON object
		largeData := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		err := j.Marshal(largeData)
		assert.NoError(t, err)

		var unmarshaled map[string]interface{}
		err = j.Unmarshal(&unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, len(largeData), len(unmarshaled))
	})

	t.Run("Base BeforeCreate with reflection", func(t *testing.T) {
		// Test that BeforeCreate works with models that embed Base at different positions
		type CustomModel struct {
			Name     string `json:"name"`
			Base     `json:",inline"`
			CustomID int `json:"custom_id"`
		}

		model := &CustomModel{}
		err := model.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, model.ID)
	})

	t.Run("Type assertions", func(t *testing.T) {
		// Test that our custom types implement the expected interfaces
		var j interface{} = JSON(`{}`)
		_, isScanner := j.(driver.Valuer)
		assert.True(t, isScanner, "JSON should implement driver.Valuer")

		var jB interface{} = JSONB(`{}`)
		_, isScannerB := jB.(driver.Valuer)
		assert.True(t, isScannerB, "JSONB should implement driver.Valuer")

		// Test Scanner interface
		var jScan interface {
			Scan(interface{}) error
		} = &JSON{}
		assert.NotNil(t, jScan, "JSON pointer should implement Scanner")

		var jBScan interface {
			Scan(interface{}) error
		} = &JSONB{}
		assert.NotNil(t, jBScan, "JSONB pointer should implement Scanner")
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	t.Run("Repeated JSON operations", func(t *testing.T) {
		var j JSON
		testData := map[string]interface{}{
			"field1": "value1",
			"field2": 42,
			"field3": true,
		}

		// Marshal 1000 times
		start := time.Now()
		for i := 0; i < 1000; i++ {
			err := j.Marshal(testData)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
		}
		duration := time.Since(start)

		// Should complete reasonably fast (adjust threshold as needed)
		assert.Less(t, duration, 100*time.Millisecond, "1000 marshals should complete quickly")
	})

	t.Run("Concurrent JSON operations", func(t *testing.T) {
		done := make(chan bool, 10)

		// Run 10 goroutines doing JSON operations
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				var j JSON
				testData := map[string]interface{}{
					"id":   id,
					"name": fmt.Sprintf("test_%d", id),
				}

				err := j.Marshal(testData)
				if err != nil {
					t.Errorf("Marshal failed in goroutine %d: %v", id, err)
					return
				}

				var result map[string]interface{}
				err = j.Unmarshal(&result)
				if err != nil {
					t.Errorf("Unmarshal failed in goroutine %d: %v", id, err)
					return
				}

				assert.Equal(t, float64(id), result["id"])
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
