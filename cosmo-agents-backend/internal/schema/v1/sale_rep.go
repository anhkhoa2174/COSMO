package v1

import (
	"time"

	"github.com/google/uuid"
)

// CreateSaleRepRequest represents the request to create a sale rep
type CreateSaleRepRequest struct {
	FirstName    string  `json:"first_name" validate:"required"`
	LastName     string  `json:"last_name" validate:"required"`
	Email        string  `json:"email" validate:"required,email"`
	CalendarLink *string `json:"calendar_link" validate:"omitempty,url"`
	Picture      *string `json:"picture" validate:"omitempty,url"`
}

// UpdateSaleRepRequest represents the request to update a sale rep
type UpdateSaleRepRequest struct {
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	CalendarLink *string `json:"calendar_link" validate:"omitempty,url"`
	Picture      *string `json:"picture" validate:"omitempty,url"`
}

// SaleRepResponse represents a sale rep entity
type SaleRepResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	CalendarLink *string   `json:"calendar_link"`
	Picture      *string   `json:"picture"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SaleRepSearchRequest represents the search request
type SaleRepSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// SaleRepListItem wraps a sale rep entity
type SaleRepListItem struct {
	Entity SaleRepResponse `json:"entity"`
}

// SaleRepSearchResponse represents paginated search results
type SaleRepSearchResponse struct {
	List   []SaleRepListItem `json:"list"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
	Total  int64             `json:"total"`
}
