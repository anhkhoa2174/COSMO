package contact

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ImportCSV handles POST /v2/contacts/import (deprecated, use V3)
func (h *Handler) ImportCSV(c fiber.Ctx) error {
	// Use auth helper
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "file is required",
		})
	}

	// Process CSV import (keeping legacy behavior)
	warnings, err := h.processCSVImport(c.Context(), fileHeader, userID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to import CSV")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to import contacts: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"warnings": warnings,
		},
	})
}

// processCSVImport processes CSV file and imports contacts
func (h *Handler) processCSVImport(ctx context.Context, file *multipart.FileHeader, userID uuid.UUID) ([]string, error) {
	// Open file
	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Read CSV
	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Process rows
	var warnings []string
	var contacts []*domain.Contact
	rowNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Row %d: %v", rowNum, err))
			rowNum++
			continue
		}

		// Build contact from row
		contact := &domain.Contact{
			Base:   domain.Base{ID: uuid.New()},
			UserID: userID,
			Source: "csv",
		}

		var email, phone string
		for i, header := range headers {
			if i >= len(record) {
				break
			}

			value := strings.TrimSpace(record[i])
			normalizedHeader := strings.ToLower(strings.TrimSpace(header))

			switch normalizedHeader {
			case "name", "full_name", "fullname":
				contact.Name = value
			case "first_name", "firstname":
				// Store temporarily for later combination
				if contact.Name == "" {
					contact.Name = value
				}
			case "last_name", "lastname":
				// Append to name if first_name was set
				if contact.Name != "" && value != "" {
					contact.Name = contact.Name + " " + value
				} else if value != "" {
					contact.Name = value
				}
			case "email":
				email = value
			case "phone":
				phone = value
			case "company":
				contact.Company = value
			case "job_title", "jobtitle":
				contact.JobTitle = value
			case "address":
				contact.Address = value
			case "city":
				contact.City = value
			case "country":
				contact.Country = value
			case "state":
				contact.State = value
			case "zip", "postal_code":
				contact.Zip = value
			}
		}

		// Store email and phone in profile
		if email != "" || phone != "" {
			profile := make(map[string]interface{})
			if email != "" {
				profile["email"] = email
			}
			if phone != "" {
				profile["phone"] = phone
			}
			_ = contact.Profile.Marshal(profile)
		}

		if email != "" {
			contact.SourceID = fmt.Sprintf("csv_%s", email)
			contacts = append(contacts, contact)
		} else {
			warnings = append(warnings, fmt.Sprintf("Row %d: missing email, skipped", rowNum))
		}

		rowNum++
	}

	// Batch upsert
	if len(contacts) > 0 {
		if err := h.contactRepo.UpsertMany(ctx, contacts); err != nil {
			return warnings, fmt.Errorf("failed to upsert contacts: %w", err)
		}
	}

	return warnings, nil
}
