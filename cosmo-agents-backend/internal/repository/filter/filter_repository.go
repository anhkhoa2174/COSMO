package filter

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"

	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// validFieldPattern ensures field names are safe for SQL
var validFieldPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidFieldName validates field name to prevent SQL injection
func isValidFieldName(field string) bool {
	return validFieldPattern.MatchString(field) && len(field) <= 64 // Reasonable length limit
}

// FilterBuilder builds GORM queries from Filter maps
// Equivalent to Python's SQLAlchemy filter building
type FilterBuilder struct {
	db *gorm.DB
}

// NewFilterBuilder creates a new filter builder
func NewFilterBuilder(db *gorm.DB) *FilterBuilder {
	return &FilterBuilder{db: db}
}

// Apply applies filters to a GORM query
func (fb *FilterBuilder) Apply(filter base.Filter) *gorm.DB {
	if len(filter) == 0 {
		return fb.db
	}

	query := fb.db

	for field, value := range filter {
		query = fb.applyField(query, field, value)
	}

	return query
}

// applyField applies a single field filter
func (fb *FilterBuilder) applyField(query *gorm.DB, field string, value interface{}) *gorm.DB {
	// Handle logical operators
	if field == string(base.OpAnd) {
		return fb.applyAnd(query, value)
	}
	if field == string(base.OpOr) {
		return fb.applyOr(query, value)
	}
	if field == string(base.OpRaw) {
		return fb.applyRaw(query, value)
	}

	// Validate field name to prevent SQL injection
	if !isValidFieldName(field) {
		// Skip invalid field names instead of erroring to maintain API compatibility
		return query
	}

	// Handle operator-based filters
	if filterMap, ok := value.(map[string]interface{}); ok {
		return fb.applyOperators(query, field, filterMap)
	}

	// Default: equality
	// Qualify field name with table if table is explicitly set
	qualifiedField := fb.qualifyField(query, field)
	return query.Where(fmt.Sprintf("%s = ?", qualifiedField), value)
}

// applyRaw supports custom raw expressions and jsonb_numeric helper
func (fb *FilterBuilder) applyRaw(query *gorm.DB, value interface{}) *gorm.DB {
	if rawExpr, ok := value.(string); ok {
		return query.Where(rawExpr)
	}
	// support map payload: {"jsonb_numeric": {"column": "scores", "path": "fit", "ops": {">=": 50}}}
	// and {"jsonb_text": {"column": "ai_insights", "path": "suspected_goals", "value": "automation"}}
	if m, ok := value.(map[string]interface{}); ok {
		if expr, ok := m["jsonb_numeric"].(map[string]interface{}); ok {
			col, _ := expr["column"].(string)
			path, _ := expr["path"].(string)
			if col == "" || path == "" {
				return query
			}
			if ops, ok := expr["ops"].(map[string]interface{}); ok {
				for op, v := range ops {
					opSQL := map[string]string{"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<"}[op]
					if opSQL == "" {
						continue
					}
					query = query.Where("(??->>?)::float "+opSQL+" ?", col, path, v)
				}
			}
		}
		if expr, ok := m["jsonb_text"].(map[string]interface{}); ok {
			col, _ := expr["column"].(string)
			path, _ := expr["path"].(string)
			val, _ := expr["value"].(string)
			if col != "" && path != "" && val != "" {
				query = query.Where("(?? #>> ?) ILIKE ?", col, "{"+path+"}", "%"+val+"%")
			}
		}
	}
	return query
}

// qualifyField adds table prefix to field name if table is set in GORM statement
func (fb *FilterBuilder) qualifyField(query *gorm.DB, field string) string {
	// If field already contains a dot (e.g., "campaigns.status"), return as-is
	if strings.Contains(field, ".") {
		return field
	}

	// Get table name from GORM statement
	stmt := query.Statement
	if stmt != nil && stmt.Table != "" {
		return fmt.Sprintf("%s.%s", stmt.Table, field)
	}

	// No table set, return bare field name
	return field
}

// applyOperators applies operator-based filters
func (fb *FilterBuilder) applyOperators(query *gorm.DB, field string, operators map[string]interface{}) *gorm.DB {
	// Validate field name to prevent SQL injection
	if !isValidFieldName(field) {
		return query
	}

	// Qualify field name with table if table is explicitly set
	qualifiedField := fb.qualifyField(query, field)

	for op, val := range operators {
		switch base.FilterOperator(op) {
		case base.OpEqual:
			query = query.Where(fmt.Sprintf("%s = ?", qualifiedField), val)

		case base.OpNotEqual:
			query = query.Where(fmt.Sprintf("%s != ?", qualifiedField), val)

		case base.OpLessThan:
			query = query.Where(fmt.Sprintf("%s < ?", qualifiedField), val)

		case base.OpLessThanEqual:
			query = query.Where(fmt.Sprintf("%s <= ?", qualifiedField), val)

		case base.OpGreaterThan:
			query = query.Where(fmt.Sprintf("%s > ?", qualifiedField), val)

		case base.OpGreaterThanEqual:
			query = query.Where(fmt.Sprintf("%s >= ?", qualifiedField), val)

		case base.OpIn:
			query = query.Where(fmt.Sprintf("%s IN ?", qualifiedField), val)

		case base.OpNotIn:
			query = query.Where(fmt.Sprintf("%s NOT IN ?", qualifiedField), val)

		case base.OpBetween:
			if arr, ok := val.([]interface{}); ok && len(arr) == 2 {
				query = query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", qualifiedField), arr[0], arr[1])
			}

		case base.OpLike:
			query = query.Where(fmt.Sprintf("%s LIKE ?", qualifiedField), val)

		case base.OpILike:
			query = query.Where(fmt.Sprintf("%s ILIKE ?", qualifiedField), val)

		case base.OpNotLike:
			query = query.Where(fmt.Sprintf("%s NOT LIKE ?", qualifiedField), val)

		case base.OpNotILike:
			query = query.Where(fmt.Sprintf("%s NOT ILIKE ?", qualifiedField), val)
		}
	}

	return query
}

// applyAnd applies AND logical operator
func (fb *FilterBuilder) applyAnd(query *gorm.DB, value interface{}) *gorm.DB {
	if conditions, ok := value.([]interface{}); ok {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				subQuery := fb.db.Session(&gorm.Session{NewDB: true})
				subFilter := NewFilterBuilder(subQuery)
				query = query.Where(subFilter.Apply(condMap))
			}
		}
	}
	return query
}

// applyOr applies OR logical operator
func (fb *FilterBuilder) applyOr(query *gorm.DB, value interface{}) *gorm.DB {
	if conditions, ok := value.([]interface{}); ok {
		orQuery := query.Session(&gorm.Session{NewDB: true})

		for i, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				subQuery := fb.db.Session(&gorm.Session{NewDB: true})
				subFilter := NewFilterBuilder(subQuery)

				if i == 0 {
					orQuery = orQuery.Where(subFilter.Apply(condMap))
				} else {
					orQuery = orQuery.Or(subFilter.Apply(condMap))
				}
			}
		}

		query = query.Where(orQuery)
	}
	return query
}

// BuildOrderBy builds ORDER BY clause from sort params
func BuildOrderBy(sortBy, sortOrder string) string {
	if sortBy == "" {
		return "created_at DESC" // Default
	}

	// Sanitize field name (prevent SQL injection)
	sortBy = strings.ReplaceAll(sortBy, ";", "")
	sortBy = strings.ReplaceAll(sortBy, "--", "")

	if sortOrder == "" || strings.ToUpper(sortOrder) == "DESC" {
		return fmt.Sprintf("%s DESC", sortBy)
	}

	return fmt.Sprintf("%s ASC", sortBy)
}
