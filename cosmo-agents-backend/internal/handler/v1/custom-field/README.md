# Custom Field Handler Package

This package implements the V1 Custom Field API endpoints following clean architecture principles.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/custom-fields` | Create a new custom field |
| GET | `/v1/custom-fields` | List all custom fields with filtering |
| PATCH | `/v1/custom-fields/:id` | Update a custom field |
| DELETE | `/v1/custom-fields/:id` | Delete a custom field |

## File Structure

```
custom-field/
├── handler.go     # Handler struct and constructor
├── helpers.go     # Shared helper functions for responses and parsing
├── create.go      # POST /v1/custom-fields - Create custom field
├── list.go        # GET /v1/custom-fields - List/filter custom fields
├── update.go      # PATCH /v1/custom-fields/:id - Update custom field
├── delete.go      # DELETE /v1/custom-fields/:id - Delete custom field
└── README.md      # This file
```

## Features

- **Ownership validation**: All operations verify user ownership before allowing modifications
- **Pagination**: List endpoint supports offset/limit parameters
- **Filtering**: List endpoint can filter by `entity_type` query parameter
- **Swagger documentation**: All endpoints have comprehensive Swagger annotations
- **Validation**: Request validation using struct tags
- **Error handling**: Consistent error responses with proper HTTP status codes
- **Nil safety**: Proper nil checks for database operations

## Usage Example

```go
// In routes setup
customFieldHandler := customfield.NewHandler(customFieldRepo, userRepo)

v1.Post("/custom-fields", customFieldHandler.Create)
v1.Get("/custom-fields", customFieldHandler.List)
v1.Patch("/custom-fields/:id", customFieldHandler.Update)
v1.Delete("/custom-fields/:id", customFieldHandler.Delete)
```

## Request/Response Examples

### Create Custom Field
```json
POST /v1/custom-fields
{
  "name": "Company Size",
  "data_type": "select",
  "entity_type": "contact",
  "is_required": false,
  "options": ["1-10", "11-50", "51-200", "201-500", "500+"]
}
```

### List Custom Fields
```
GET /v1/custom-fields?entity_type=contact&offset=0&limit=25
```

### Update Custom Field
```json
PATCH /v1/custom-fields/{id}
{
  "name": "Company Size Range",
  "is_required": true
}
```

## Dependencies

- **Repository**: `repository.CustomFieldRepository` - Database operations
- **Schema**: `schema/v1.CustomFieldRequest/Response` - Request/response types
- **Domain**: `domain.CustomField` - Domain model
