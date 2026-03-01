package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterOperator_String(t *testing.T) {
	tests := []struct {
		name     string
		operator FilterOperator
		expected string
	}{
		{"Equal", OpEqual, "="},
		{"NotEqual", OpNotEqual, "!="},
		{"LessThan", OpLessThan, "<"},
		{"LessThanEqual", OpLessThanEqual, "<="},
		{"GreaterThan", OpGreaterThan, ">"},
		{"GreaterThanEqual", OpGreaterThanEqual, ">="},
		{"In", OpIn, "in"},
		{"NotIn", OpNotIn, "not_in"},
		{"Between", OpBetween, "between"},
		{"Like", OpLike, "like"},
		{"ILike", OpILike, "ilike"},
		{"NotLike", OpNotLike, "not_like"},
		{"NotILike", OpNotILike, "not_ilike"},
		{"And", OpAnd, "and"},
		{"Or", OpOr, "or"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.operator.String())
		})
	}
}

func TestPaginationParams_Validate(t *testing.T) {
	tests := []struct {
		name      string
		params    PaginationParams
		expected  PaginationParams
		shouldErr bool
	}{
		{
			name:      "valid params",
			params:    PaginationParams{Limit: 10, Offset: 0},
			expected:  PaginationParams{Limit: 10, Offset: 0},
			shouldErr: false,
		},
		{
			name:      "zero limit should use default",
			params:    PaginationParams{Limit: 0, Offset: 0},
			expected:  PaginationParams{Limit: DefaultPaginationLimit, Offset: 0},
			shouldErr: false,
		},
		{
			name:      "negative limit should use default",
			params:    PaginationParams{Limit: -5, Offset: 0},
			expected:  PaginationParams{Limit: DefaultPaginationLimit, Offset: 0},
			shouldErr: false,
		},
		{
			name:      "limit too large should cap",
			params:    PaginationParams{Limit: 1000, Offset: 0},
			expected:  PaginationParams{Limit: DefaultPaginationLimit, Offset: 0},
			shouldErr: false,
		},
		{
			name:      "negative offset should be zero",
			params:    PaginationParams{Limit: 10, Offset: -5},
			expected:  PaginationParams{Limit: 10, Offset: 0},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.params
			params.Validate()

			assert.Equal(t, tt.expected.Limit, params.Limit)
			assert.Equal(t, tt.expected.Offset, params.Offset)
		})
	}
}

func TestDefaultPagination(t *testing.T) {
	pagination := DefaultPagination()

	assert.Equal(t, DefaultPaginationLimit, pagination.Limit)
	assert.Equal(t, int(0), pagination.Offset)
}

func TestBuildOrderBy(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		expected  string
	}{
		{
			name:      "empty sort by",
			sortBy:    "",
			sortOrder: "",
			expected:  "created_at DESC",
		},
		{
			name:      "sort by with default sort order",
			sortBy:    "name",
			sortOrder: "",
			expected:  "name DESC",
		},
		{
			name:      "sort by with uppercase desc",
			sortBy:    "age",
			sortOrder: "DESC",
			expected:  "age DESC",
		},
		{
			name:      "sort by with lowercase desc",
			sortBy:    "age",
			sortOrder: "desc",
			expected:  "age DESC",
		},
		{
			name:      "sort by with uppercase asc",
			sortBy:    "email",
			sortOrder: "ASC",
			expected:  "email ASC",
		},
		{
			name:      "sort by with mixed case",
			sortBy:    "status",
			sortOrder: "AsC",
			expected:  "status ASC",
		},
		{
			name:      "SQL injection attempt with semicolon",
			sortBy:    "name; DROP TABLE users; --",
			sortOrder: "ASC",
			expected:  "name DROP TABLE users   ASC", // Should be sanitized
		},
		{
			name:      "SQL injection attempt with double dash",
			sortBy:    "name--",
			sortOrder: "DESC",
			expected:  "name DESC", // Should be sanitized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildOrderBy(tt.sortBy, tt.sortOrder)
			assert.Equal(t, tt.expected, result)
		})
	}
}
