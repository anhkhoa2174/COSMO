package contact

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// CSVImporter handles CSV file import for contacts
type CSVImporter struct {
	contactRepo ContactRepositoryInterface
}

// NewCSVImporter creates a new CSVImporter
func NewCSVImporter(contactRepo ContactRepositoryInterface) *CSVImporter {
	return &CSVImporter{
		contactRepo: contactRepo,
	}
}

// ImportConfig contains configuration for CSV import
type ImportConfig struct {
	FilePath       string
	UserID         uuid.UUID
	OrganizationID *uuid.UUID
	RequiredFields []string
	// FieldMapping maps CSV column names to contact field names
	// e.g., {"Company Name": "company", "Job": "job_title", "Custom Field 1": "custom_field_1"}
	FieldMapping map[string]string
	// ExplicitMapping imports only the columns in FieldMapping. RequiredFields
	// are then contact fields that must be mapped to a column, not header
	// names, and a mapped field that is not a contact column is stored in
	// profile under its own name. Without it, every unmapped column is
	// auto-mapped, so a column that happens to be called "email" competes with
	// the one the user picked for email.
	ExplicitMapping bool
}

// ImportResult contains the result of a CSV import operation
type ImportResult struct {
	TotalRows       int
	ImportedRows    int
	SkippedRows     int
	RejectedRows    int               // Contacts rejected due to ownership conflict
	RejectedReasons map[string]string // Map of email/identifier -> reason
	ErrorMessages   []string
}

// Import processes a CSV file and imports contacts
func (s *CSVImporter) Import(ctx context.Context, config ImportConfig) (*ImportResult, error) {
	result := &ImportResult{}

	// Set default required fields if not provided
	if len(config.RequiredFields) == 0 {
		config.RequiredFields = []string{"first_name", "last_name", "email"}
	}

	// Step 1: Open and read CSV file
	file, err := os.Open(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Step 2: Parse headers. Excel's "CSV UTF-8" export prefixes a byte order
	// mark, which TrimSpace keeps; left on, the first column never matched
	// its name (a leading "name" column landed in profile as "\ufeffname").
	headers := records[0]
	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = i
	}

	// Step 3: Validate required columns
	if config.ExplicitMapping {
		mapped := make(map[string]bool, len(config.FieldMapping))
		for column, target := range config.FieldMapping {
			if _, exists := headerMap[strings.TrimSpace(strings.ToLower(column))]; !exists {
				return nil, fmt.Errorf("column '%s' is not in the CSV file", column)
			}
			mapped[strings.ToLower(strings.TrimSpace(target))] = true
		}
		for _, field := range config.RequiredFields {
			if !mapped[field] {
				return nil, fmt.Errorf("no column is mapped to required field '%s'", field)
			}
		}
	} else {
		for _, field := range config.RequiredFields {
			if _, exists := headerMap[field]; !exists {
				return nil, fmt.Errorf("missing required header '%s'", field)
			}
		}
	}

	// Step 4: Check if we have data rows
	if len(records) < 2 {
		result.TotalRows = 0
		return result, nil
	}

	result.TotalRows = len(records) - 1
	result.RejectedReasons = make(map[string]string)

	// Step 5: Build effective field mapping (CSV column -> contact field)
	var effectiveMapping map[string]string
	if config.ExplicitMapping {
		effectiveMapping = explicitMapping(config.FieldMapping)
	} else {
		effectiveMapping = s.buildEffectiveMapping(headers, config.FieldMapping)
	}

	// Step 6: Process each row and create contacts
	contacts, skipped := s.parseCSVRecordsWithMapping(records, headerMap, effectiveMapping, config.UserID, config.OrganizationID)
	result.SkippedRows = skipped

	// Step 7: Batch upsert contacts with ownership check
	if len(contacts) > 0 {
		if config.OrganizationID != nil {
			// Use ownership check when organization is specified
			ownershipResult, err := s.contactRepo.UpsertManyWithOwnershipCheck(ctx, contacts, config.UserID, *config.OrganizationID)
			if err != nil {
				return result, fmt.Errorf("failed to upsert contacts: %w", err)
			}
			result.ImportedRows = len(ownershipResult.ValidContacts)
			result.RejectedRows = len(ownershipResult.RejectedContacts)
			result.RejectedReasons = ownershipResult.RejectedReasons
		} else {
			// No organization, use regular upsert
			err = s.contactRepo.UpsertMany(ctx, contacts)
			if err != nil {
				return result, fmt.Errorf("failed to upsert contacts: %w", err)
			}
			result.ImportedRows = len(contacts)
		}
	}

	return result, nil
}

// Standard contact fields that map directly to Contact struct
var standardFields = map[string]bool{
	// "name" is read by combineName; without it here a "name" column was
	// mapped to profile.name and every contact was saved as "N/A".
	"name":       true,
	"first_name": true,
	"last_name":  true,
	"email":      true,
	"phone":      true,
	"company":    true,
	"job_title":  true,
	"address":    true,
	"city":       true,
	"country":    true,
	"state":      true,
	"zip":        true,
	// Contact columns too; before they were listed here an "industry" column
	// was filed under profile and the contact's own field stayed empty.
	"industry":        true,
	"contact_channel": true,
}

// explicitMapping normalises a user-chosen column -> field mapping into the
// form parseCSVRecordsWithMapping reads: lower-cased column names, and fields
// that are not contact columns redirected into profile.
func explicitMapping(fieldMapping map[string]string) map[string]string {
	out := make(map[string]string, len(fieldMapping))
	for column, target := range fieldMapping {
		column = strings.TrimSpace(strings.ToLower(column))
		target = strings.TrimSpace(strings.ToLower(target))
		if column == "" || target == "" {
			continue
		}
		if !standardFields[target] && !strings.HasPrefix(target, "profile.") {
			target = "profile." + target
		}
		out[column] = target
	}
	return out
}

// buildEffectiveMapping creates the final field mapping from CSV headers
// It uses provided mapping or auto-maps matching headers to standard fields
func (s *CSVImporter) buildEffectiveMapping(headers []string, customMapping map[string]string) map[string]string {
	effectiveMapping := make(map[string]string)

	for _, header := range headers {
		normalizedHeader := strings.TrimSpace(strings.ToLower(header))

		// Check if there's a custom mapping for this header
		if customMapping != nil {
			if targetField, exists := customMapping[header]; exists {
				effectiveMapping[normalizedHeader] = strings.ToLower(strings.TrimSpace(targetField))
				continue
			}
			// Also check normalized header in custom mapping
			if targetField, exists := customMapping[normalizedHeader]; exists {
				effectiveMapping[normalizedHeader] = strings.ToLower(strings.TrimSpace(targetField))
				continue
			}
		}

		// Auto-map if header matches a standard field name
		if standardFields[normalizedHeader] {
			effectiveMapping[normalizedHeader] = normalizedHeader
		} else {
			// Non-standard fields go to profile (custom fields)
			// Normalize the field name for storage
			normalizedFieldName := strings.ReplaceAll(normalizedHeader, " ", "_")
			effectiveMapping[normalizedHeader] = "profile." + normalizedFieldName
		}
	}

	return effectiveMapping
}

// parseCSVRecordsWithMapping converts CSV records to domain.Contact objects using field mapping
func (s *CSVImporter) parseCSVRecordsWithMapping(records [][]string, headerMap map[string]int, fieldMapping map[string]string, userID uuid.UUID, orgID *uuid.UUID) ([]*domain.Contact, int) {
	contacts := make([]*domain.Contact, 0, len(records)-1)
	// One INSERT ... ON CONFLICT cannot touch the same row twice, so a file
	// repeating an email failed as a whole; the later row wins instead.
	bySourceID := make(map[string]int)
	skipped := 0
	now := time.Now().UTC()

	for i := 1; i < len(records); i++ {
		row := records[i]

		// Skip empty rows
		if s.isEmptyRow(row) {
			skipped++
			continue
		}

		// Get field value by CSV header name
		getFieldByHeader := func(headerName string, defaultValue string) string {
			if idx, exists := headerMap[headerName]; exists && idx < len(row) {
				value := strings.TrimSpace(row[idx])
				if value == "" {
					return defaultValue
				}
				return value
			}
			return defaultValue
		}

		// Get field value by target field name (looks up which CSV header maps to this field)
		getFieldByTarget := func(targetField string, defaultValue string) string {
			for header, target := range fieldMapping {
				if target == targetField {
					return getFieldByHeader(header, defaultValue)
				}
			}
			return defaultValue
		}

		email := getFieldByTarget("email", "N/A")

		// Build profile (custom fields) map
		profile := make(map[string]interface{})
		for header, target := range fieldMapping {
			if strings.HasPrefix(target, "profile.") {
				fieldName := strings.TrimPrefix(target, "profile.")
				value := getFieldByHeader(header, "")
				if value != "" {
					profile[fieldName] = value
				}
			}
		}

		// Add email and phone to profile (they're now stored in profile JSONB)
		if email != "" && email != "N/A" {
			profile["email"] = email
		}
		phone := getFieldByTarget("phone", "N/A")
		if phone != "" && phone != "N/A" {
			profile["phone"] = phone
		}

		// Create contact
		contact := &domain.Contact{
			Base: domain.Base{
				ID: uuid.New(),
			},
			UserID:         userID,
			OrganizationID: orgID,
			Source:         string(domain.ContactSourceCSV),
			SourceID:       csvSourceID(email),
			Name:           combineName(getFieldByTarget("first_name", ""), getFieldByTarget("last_name", ""), getFieldByTarget("name", "N/A")),
			Company:        getFieldByTarget("company", "N/A"),
			JobTitle:       getFieldByTarget("job_title", "N/A"),
			Address:        getFieldByTarget("address", "N/A"),
			City:           getFieldByTarget("city", "N/A"),
			Country:        getFieldByTarget("country", "N/A"),
			State:          getFieldByTarget("state", "N/A"),
			Zip:            getFieldByTarget("zip", "N/A"),
			Industry:       getFieldByTarget("industry", ""),
			ContactChannel: getFieldByTarget("contact_channel", ""),
			DoNotContact:   false,
			TimestampMixin: domain.TimestampMixin{
				CreatedAt: now,
				UpdatedAt: now,
			},
			SoftDeleteMixin: domain.SoftDeleteMixin{
				IsDeleted: false,
			},
		}

		// Set profile with email, phone, and custom fields
		if len(profile) > 0 {
			if profileBytes, err := json.Marshal(profile); err == nil {
				contact.Profile = profileBytes
			}
		}

		// Create sets these too: contact_information is the duplicate key and
		// the outreach recipient, and without a status every imported row
		// stayed "pending" even when complete.
		if email != "" && email != "N/A" {
			contact.ContactInformation = email
		}
		contact.CalculateStatus()

		if prev, dup := bySourceID[contact.SourceID]; dup {
			contacts[prev] = contact
			skipped++
			continue
		}
		bySourceID[contact.SourceID] = len(contacts)
		contacts = append(contacts, contact)
	}

	return contacts, skipped
}

// csvSourceID keys an imported row by its email, case-folded so that
// "Ann@x.com" and "ann@x.com" are one contact. A row without an email gets a
// key of its own: sharing "csv_N/A" collapsed all of them into one contact.
func csvSourceID(email string) string {
	if email == "" || email == "N/A" {
		return fmt.Sprintf("csv_%s", uuid.NewString())
	}
	return fmt.Sprintf("csv_%s", strings.ToLower(email))
}

// parseCSVRecords converts CSV records to domain.Contact objects (legacy method for backwards compatibility)
func (s *CSVImporter) parseCSVRecords(records [][]string, headerMap map[string]int, userID uuid.UUID) ([]*domain.Contact, int) {
	contacts := make([]*domain.Contact, 0, len(records)-1)
	skipped := 0
	now := time.Now().UTC()

	for i := 1; i < len(records); i++ {
		row := records[i]

		// Skip empty rows
		if s.isEmptyRow(row) {
			skipped++
			continue
		}

		// Get field values safely
		getField := func(fieldName string, defaultValue string) string {
			if idx, exists := headerMap[fieldName]; exists && idx < len(row) {
				value := strings.TrimSpace(row[idx])
				if value == "" {
					return defaultValue
				}
				return value
			}
			return defaultValue
		}

		email := getField("email", "N/A")
		phone := getField("phone", "N/A")

		// Build profile with email and phone
		profile := make(map[string]interface{})
		if email != "" && email != "N/A" {
			profile["email"] = email
		}
		if phone != "" && phone != "N/A" {
			profile["phone"] = phone
		}
		var profileBytes domain.JSONB
		if len(profile) > 0 {
			_ = profileBytes.Marshal(profile)
		}

		// Create contact
		contact := &domain.Contact{
			Base: domain.Base{
				ID: uuid.New(),
			},
			UserID:       userID,
			Source:       string(domain.ContactSourceCSV),
			SourceID:     fmt.Sprintf("csv_%s", email),
			Name:         combineName(getField("first_name", ""), getField("last_name", ""), getField("name", "N/A")),
			Profile:      profileBytes,
			Company:      getField("company", "N/A"),
			JobTitle:     getField("job_title", "N/A"),
			Address:      getField("address", "N/A"),
			City:         getField("city", "N/A"),
			Country:      getField("country", "N/A"),
			State:        getField("state", "N/A"),
			Zip:          getField("zip", "N/A"),
			DoNotContact: false,
			TimestampMixin: domain.TimestampMixin{
				CreatedAt: now,
				UpdatedAt: now,
			},
			SoftDeleteMixin: domain.SoftDeleteMixin{
				IsDeleted: false,
			},
		}

		contacts = append(contacts, contact)
	}

	return contacts, skipped
}

// isEmptyRow checks if a CSV row is empty
func (s *CSVImporter) isEmptyRow(row []string) bool {
	if len(row) == 0 {
		return true
	}
	if len(row) == 1 && strings.TrimSpace(row[0]) == "" {
		return true
	}
	// Check if all elements in the row are empty or whitespace
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

// combineName combines first name and last name into a single name field
func combineName(firstName, lastName, fallback string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	if fallback != "" {
		return fallback
	}
	return "N/A"
}

// ValidateCSVFile validates CSV file structure without importing
func (s *CSVImporter) ValidateCSVFile(filePath string, requiredFields []string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}
	headerMap := make(map[string]bool)
	for _, header := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = true
	}

	for _, field := range requiredFields {
		if !headerMap[field] {
			return fmt.Errorf("missing required header '%s'", field)
		}
	}

	return nil
}
