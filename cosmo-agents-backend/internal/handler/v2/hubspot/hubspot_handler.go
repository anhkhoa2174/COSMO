package hubspot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	integrationRepo "github.com/rockship/cosmo-agents-go/internal/repository/integration"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/hubspot"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ErrNoChanges indicates no changes were needed
var ErrNoChanges = errors.New("no changes needed")

// ErrIntegrationNotFound indicates integration was not found
var ErrIntegrationNotFound = errors.New("integration not found")

// HubspotHandler handles V2 HubSpot requests
type HubspotHandler struct {
	session         core.Session
	integrationRepo *integrationRepo.IntegrationRepository
	userRepo        *user.UserRepository
	hubspotClient   *hubspot.Client
}

// NewHubspotHandler creates a new V2 HubspotHandler
func NewHubspotHandler(
	session core.Session,
	integrationRepo *integrationRepo.IntegrationRepository,
	userRepo *user.UserRepository,
	hubspotClient *hubspot.Client,
) *HubspotHandler {
	return &HubspotHandler{
		session:         session,
		integrationRepo: integrationRepo,
		userRepo:        userRepo,
		hubspotClient:   hubspotClient,
	}
}

// executeInTransaction executes a function within a database transaction
// Ensures atomic operations: rollback on error/panic, commit on success
// Using mentor's pattern with core.Session
func (h *HubspotHandler) executeInTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) (err error) {
	session, err := h.session.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			// The rollback uses the session's stored transaction context
			if rbErr := session.Rollback(); rbErr != nil {
				logger.Logger.Error().Err(rbErr).Msg("rollback failed during panic recovery")
				err = fmt.Errorf("panic during transaction: %v; rollback failed: %v", r, rbErr)
				return
			}
			err = fmt.Errorf("panic during transaction: %v", r)
			return
		}
	}()

	if err = fn(session.Context()); err != nil {
		// The rollback uses the session's stored transaction context
		if rbErr := session.Rollback(); rbErr != nil {
			logger.Logger.Error().
				Err(rbErr).
				Msg("rollback failed after transaction error")
			return fmt.Errorf("transaction failed: %w; rollback failed: %v", err, rbErr)
		}
		return err
	}

	if err = session.Commit(); err != nil {
		logger.Logger.Error().Err(err).Msg("session commit failed")
		return err
	}

	return nil
}

// mapsEqual compares two map[string]interface{} values for equality
// Optimized for HubSpot field mapping comparison with fast path checks and optimized value comparison
func mapsEqual(a, b interface{}) bool {
	// Fast path: same reference
	if a == b {
		return true
	}

	// Type assertion to map[string]interface{}
	mapA, okA := a.(map[string]interface{})
	mapB, okB := b.(map[string]interface{})

	// Handle nil cases
	if !okA || !okB {
		// Fall back to reflect.DeepEqual for non-map types
		return reflect.DeepEqual(a, b)
	}

	// Fast path: different lengths
	if len(mapA) != len(mapB) {
		return false
	}

	// Compare each key-value pair with optimized comparison for common types
	for key, valA := range mapA {
		valB, exists := mapB[key]
		if !exists {
			return false
		}

		// Optimized comparison for common HubSpot field mapping types
		if !valuesEqual(valA, valB) {
			return false
		}
	}

	return true
}

// valuesEqual compares two interface{} values with optimization for common types
// Avoids reflect.DeepEqual overhead for simple types commonly found in field mappings
func valuesEqual(a, b interface{}) bool {
	// Fast path: same reference
	if a == b {
		return true
	}

	// Handle nil cases
	if a == nil || b == nil {
		return a == b
	}

	// Optimized comparison for common types in HubSpot field mappings
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case int, int32, int64, float32, float64, bool:
		// For numeric and boolean types, use type assertion and direct comparison
		switch vb := b.(type) {
		case string:
			// String vs number/boolean comparison - not equal
			return false
		case int:
			if ia, ok := a.(int); ok {
				return ia == vb
			}
		case int32:
			if ia, ok := a.(int32); ok {
				return ia == vb
			}
		case int64:
			if ia, ok := a.(int64); ok {
				return ia == vb
			}
		case float32:
			if ia, ok := a.(float32); ok {
				return ia == vb
			}
		case float64:
			if ia, ok := a.(float64); ok {
				return ia == vb
			}
		case bool:
			if ia, ok := a.(bool); ok {
				return ia == vb
			}
		}
	case []interface{}:
		if vb, ok := b.([]interface{}); ok {
			return slicesEqual(va, vb)
		}
	case map[string]interface{}:
		if vb, ok := b.(map[string]interface{}); ok {
			return mapsEqual(va, vb) // Recursive call for nested maps
		}
	}

	// Fall back to reflect.DeepEqual for complex types and edge cases
	return reflect.DeepEqual(a, b)
}

// slicesEqual compares two []interface{} slices for equality
func slicesEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, va := range a {
		if !valuesEqual(va, b[i]) {
			return false
		}
	}
	return true
}

// upsertUserHubspotCredentials writes tokens to users.hubspot_credentials (parity with v1/Python).
// Preserves existing refresh_token if provider doesn't return a new one.
func (h *HubspotHandler) upsertUserHubspotCredentials(ctx context.Context, userID uuid.UUID, accessToken, refreshToken, hubID string) error {
	creds := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"hub_id":        hubID,
	}

	// If refresh token missing, try to reuse existing one
	if refreshToken == "" {
		user, err := h.userRepo.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to load user for hubspot creds: %w", err)
		}
		if user != nil && user.HubspotCredentials != nil {
			var existing map[string]interface{}
			if err := user.HubspotCredentials.Unmarshal(&existing); err == nil {
				if val, ok := existing["refresh_token"].(string); ok && val != "" {
					creds["refresh_token"] = val
				}
			}
		}
	}

	var serialized domain.JSON
	if err := serialized.Marshal(creds); err != nil {
		return fmt.Errorf("failed to serialize hubspot credentials: %w", err)
	}

	if err := h.userRepo.UpdateHubspotCredentials(ctx, userID, serialized); err != nil {
		return fmt.Errorf("failed to update user hubspot credentials: %w", err)
	}
	return nil
}

// Callback godoc
// @Summary HubSpot OAuth callback
// @Description Handles the HubSpot OAuth callback and exchanges code for tokens
// @Tags HubSpot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param code query string true "Authorization code from HubSpot"
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} map[string]interface{} "HubSpot tokens"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /v2/hubspot/callback [get]
func (h *HubspotHandler) Callback(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	code := c.Query("code")
	// redirectURI := c.Query("redirect_uri") // Not used in current implementation

	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Missing authorization code",
		})
	}

	// Exchange code for tokens
	tokenResp, err := h.hubspotClient.ExchangeCode(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to exchange code: %v", err),
		})
	}

	// Get user info from refresh token
	userInfo, err := h.hubspotClient.GetUserInfoFromRefreshToken(c.Context(), tokenResp.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to get user info: %v", err),
		})
	}

	// Store credentials and config
	credentialMap := map[string]interface{}{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
	}

	configMap := map[string]interface{}{
		"user": userInfo.User,
	}

	// Convert to JSONB
	credentialBytes, err := json.Marshal(credentialMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to encode credentials: %v", err),
		})
	}
	configBytes, err := json.Marshal(configMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to encode config: %v", err),
		})
	}

	// Add timeout context for entire transaction to prevent long-running operations
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	txErr := h.executeInTransaction(ctx, func(txCtx context.Context) error {
		// Check context cancellation before attempting lock acquisition
		if err := txCtx.Err(); err != nil {
			return fmt.Errorf("context cancelled before OAuth lock acquisition: %w", err)
		}

		// Use the transaction context for lock acquisition (inherits timeout from original context)
		current, err := h.integrationRepo.FindByUserIDAndSourceForUpdate(txCtx, userID, domain.SourceIntegrationHubspot)
		if err != nil {
			if errors.Is(err, integrationRepo.ErrLockTimeout) {
				return integrationRepo.ErrLockTimeout // Return lock timeout error for handler to process
			}
			return fmt.Errorf("failed to query integration: %w", err)
		}

		if current == nil {
			newIntegration := &domain.Integration{
				ID:         uuid.New().String(),
				UserID:     userID,
				Source:     domain.SourceIntegrationHubspot,
				SourceID:   fmt.Sprintf("%d", userInfo.HubID),
				Credential: domain.JSONB(credentialBytes),
				Config:     domain.JSONB(configBytes),
			}

			if _, err := h.integrationRepo.Create(txCtx, newIntegration); err != nil {
				return fmt.Errorf("failed to create integration: %w", err)
			}

			// Also persist credentials to users.hubspot_credentials for parity with v1/Python
			if err := h.upsertUserHubspotCredentials(txCtx, userID, tokenResp.AccessToken, tokenResp.RefreshToken, fmt.Sprintf("%d", userInfo.HubID)); err != nil {
				return err
			}
			return nil
		}

		if err := h.integrationRepo.UpdateCredential(txCtx, current.ID, credentialMap); err != nil {
			return fmt.Errorf("failed to update credentials: %w", err)
		}

		if current.Config != nil && len(current.Config) > 0 {
			var existingConfig map[string]interface{}
			if err := json.Unmarshal(current.Config, &existingConfig); err != nil {
				return fmt.Errorf("failed to parse existing config: %w", err)
			}
			for k, v := range existingConfig {
				if _, exists := configMap[k]; !exists {
					configMap[k] = v
				}
			}
		}

		if err := h.integrationRepo.UpdateConfig(txCtx, current.ID, configMap); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}

		// Update user hubspot_credentials in parallel with integration updates
		if err := h.upsertUserHubspotCredentials(txCtx, userID, tokenResp.AccessToken, tokenResp.RefreshToken, current.SourceID); err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, integrationRepo.ErrLockTimeout) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"status":  "error",
				"message": "Another HubSpot integration is in progress. Please try again.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to upsert integration: %v", txErr),
		})
	}

	response := v2schema.AuthHubspotCallbackResponse{
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		AccessToken:  tokenResp.AccessToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}

// GetUser retrieves the current user's HubSpot integration info
// GET /v2/hubspot/users/me
// GetUser godoc
// @Summary Get HubSpot user info
// @Description Get the current user's HubSpot integration information
// @Tags HubSpot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "HubSpot user info"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Not found"
// @Router /v2/hubspot/users/me [get]
func (h *HubspotHandler) GetUser(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Find integration
	integration, err := h.integrationRepo.FindByUserIDAndSource(c.Context(), userID, domain.SourceIntegrationHubspot)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to query integration",
		})
	}

	if integration == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "HubSpot integration not found",
		})
	}

	// Parse UUID from integration ID
	integrationUUID, err := uuid.Parse(integration.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid integration ID",
		})
	}

	// Parse config and credential from JSONB
	var config map[string]interface{}
	if len(integration.Config) > 0 {
		if err := json.Unmarshal(integration.Config, &config); err != nil {
			logger.Logger.Error().Err(err).Msg("failed to parse hubspot config")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to parse integration config",
			})
		}
	}

	var credential map[string]interface{}
	if len(integration.Credential) > 0 {
		if err := json.Unmarshal(integration.Credential, &credential); err != nil {
			logger.Logger.Error().Err(err).Msg("failed to parse hubspot credential")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to parse integration credential",
			})
		}
	}

	// Extract values from config and credential
	email := ""
	if config != nil {
		if user, ok := config["user"].(string); ok {
			email = user
		}
	}

	accessToken := ""
	if credential != nil {
		if token, ok := credential["access_token"].(string); ok {
			accessToken = token
		}
	}

	refreshToken := ""
	if credential != nil {
		if token, ok := credential["refresh_token"].(string); ok {
			refreshToken = token
		}
	}

	var fieldMapping map[string]interface{}
	if config != nil {
		if fm, ok := config["field_mapping"]; ok {
			if fmMap, ok := fm.(map[string]interface{}); ok {
				fieldMapping = fmMap
			} else {
				// Try to unmarshal if it's JSON
				if fmBytes, err := json.Marshal(fm); err == nil {
					if err := json.Unmarshal(fmBytes, &fieldMapping); err != nil {
						// Log error and set fieldMapping to empty map to prevent nil pointer issues
						logger.Logger.Error().Err(err).Msg("failed to parse field_mapping")
						fieldMapping = make(map[string]interface{}) // Set to empty map as fallback
					}
				} else {
					// Failed to marshal, set to empty map
					logger.Logger.Error().Err(err).Msg("failed to marshal field_mapping for parsing")
					fieldMapping = make(map[string]interface{})
				}
			}
		}
	}

	var duplicationOption *string
	if config != nil {
		if do, ok := config["duplication_option"].(string); ok {
			duplicationOption = &do
		}
	}

	response := v2schema.HubspotUserInfoRead{
		ID:                integrationUUID,
		Email:             email,
		SourceID:          integration.SourceID,
		AccessToken:       accessToken,
		RefreshToken:      refreshToken,
		FieldMapping:      fieldMapping,
		DuplicationOption: duplicationOption,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}

// UpdateUser godoc
// @Summary Update HubSpot configuration
// @Description Update the current user's HubSpot integration settings
// @Tags HubSpot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param config body map[string]interface{} true "HubSpot configuration"
// @Success 200 {object} map[string]interface{} "Update successful"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /v2/hubspot/users/me [patch]
func (h *HubspotHandler) UpdateUser(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse request body
	var req v2schema.HubspotUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	// Add timeout context for entire transaction to prevent long-running operations
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	// Update only provided fields within transaction for consistency
	err := h.executeInTransaction(ctx, func(txCtx context.Context) error {
		// Check context cancellation before attempting lock acquisition
		if err := txCtx.Err(); err != nil {
			return fmt.Errorf("context cancelled before update lock acquisition: %w", err)
		}

		// Use the transaction context for lock acquisition (inherits timeout from original context)
		latest, err := h.integrationRepo.FindByUserIDAndSourceForUpdate(txCtx, userID, domain.SourceIntegrationHubspot)
		if err != nil {
			if errors.Is(err, integrationRepo.ErrLockTimeout) {
				return integrationRepo.ErrLockTimeout // Return lock timeout error for handler to process
			}
			return fmt.Errorf("failed to reload integration with lock: %w", err)
		}

		if latest == nil {
			return ErrIntegrationNotFound
		}

		var configMap map[string]interface{}
		if len(latest.Config) > 0 {
			if err := json.Unmarshal(latest.Config, &configMap); err != nil {
				return fmt.Errorf("failed to parse hubspot config: %w", err)
			}
		}
		if configMap == nil {
			configMap = make(map[string]interface{})
		}

		// Track if we made any changes
		hasChanges := false

		if req.FieldMapping != nil {
			existingFieldMapping, exists := configMap["field_mapping"]
			// CRITICAL FIX: Use reflect.DeepEqual instead of mapsEqual to prevent panic
			// mapsEqual can panic when comparing different map types with map[string]interface{}
			if !exists || !reflect.DeepEqual(req.FieldMapping, existingFieldMapping) {
				configMap["field_mapping"] = req.FieldMapping
				hasChanges = true
			}
		}

		if req.DuplicationOption != nil {
			existingDupOption, exists := configMap["duplication_option"]
			if !exists {
				configMap["duplication_option"] = *req.DuplicationOption
				hasChanges = true
			} else {
				// Safe type assertion to prevent panic
				if existingStr, ok := existingDupOption.(string); ok {
					if *req.DuplicationOption != existingStr {
						configMap["duplication_option"] = *req.DuplicationOption
						hasChanges = true
					}
				} else {
					// Type is different, update it
					configMap["duplication_option"] = *req.DuplicationOption
					hasChanges = true
				}
			}
		}

		// Return custom error if no changes to distinguish from success
		if !hasChanges {
			return ErrNoChanges
		}

		// Update config in database within transaction
		if err := h.integrationRepo.UpdateConfig(txCtx, latest.ID, configMap); err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrNoChanges) {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "success",
				"message": "No changes needed - configuration already up to date",
				"updated": false,
			})
		}
		if errors.Is(err, ErrIntegrationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "HubSpot integration not found",
			})
		}
		if errors.Is(err, integrationRepo.ErrLockTimeout) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"status":  "error",
				"message": "Another update is in progress. Please try again.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to update configuration: %v", err),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "HubSpot configuration updated successfully",
		"updated": true,
	})
}
