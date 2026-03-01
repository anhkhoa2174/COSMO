package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	repositoryHelper "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// AuthTokens represents the token bundle returned to clients (parity with Python).
type AuthTokens struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
}

// InviteMemberResult wraps the outcome of invite-member callback handling.
type InviteMemberResult struct {
	AuthorizationURL string
	Tokens           *AuthTokens
	User             *domain.User
}

// GoogleAuthService mirrors machine.services.v2.auth.GoogleAuth logic.
type GoogleAuthService struct {
	userRepo    *userRepo.UserRepository
	roleRepo    *roleRepo.RoleRepository
	jwtManager  *auth.JWTManager
	oauthClient *googleoauth.Client
}

// NewGoogleAuthService constructs a GoogleAuthService.
func NewGoogleAuthService(
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	jwtManager *auth.JWTManager,
	oauthClient *googleoauth.Client,
) *GoogleAuthService {
	return &GoogleAuthService{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		jwtManager:  jwtManager,
		oauthClient: oauthClient,
	}
}

// GetAuthorizationURL generates the Google OAuth authorization URL.
func (s *GoogleAuthService) GetAuthorizationURL(
	ctx context.Context,
	redirectURI string,
	state string,
	loginHint string,
) (string, error) {
	if s.oauthClient == nil {
		return "", errors.New("google oauth client not configured")
	}
	cfg := s.oauthClient.GetConfig()
	if cfg == nil {
		return "", errors.New("google oauth config is not available")
	}

	if state == "" {
		state = uuid.NewString()
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("prompt", "consent"),
	}
	if loginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", loginHint))
	}
	if redirectURI != "" {
		opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", redirectURI))
	}

	return cfg.AuthCodeURL(state, opts...), nil
}

// HandleOAuth2Callback exchanges the authorization code, upserts the user, and returns tokens.
func (s *GoogleAuthService) HandleOAuth2Callback(
	ctx context.Context,
	authorizationResponse string,
	state string,
	redirectURI string,
) (*domain.User, *AuthTokens, error) {
	if s.oauthClient == nil {
		return nil, nil, errors.New("google oauth client not configured")
	}
	code, err := extractQueryParam(authorizationResponse, "code")
	if err != nil || code == "" {
		return nil, nil, errors.New("missing authorization code")
	}

	client, err := s.clientForRedirect(redirectURI)
	if err != nil {
		return nil, nil, err
	}

	token, err := client.ExchangeCode(ctx, code)
	if err != nil {
		logger.Logger.Error().
			Err(err).
			Str("redirect_uri", redirectURI).
			Msg("Google OAuth code exchange failed")
		return nil, nil, errors.New("failed to exchange authorization code")
	}

	userInfo, err := client.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		logger.Logger.Error().
			Err(err).
			Msg("Google OAuth user info retrieval failed")
		return nil, nil, errors.New("failed to fetch user information")
	}

	user, err := s.upsertGoogleUser(ctx, userInfo, token, client.GetConfig())
	if err != nil {
		return nil, nil, err
	}

	fullUser, err := s.userRepo.FindWithOrganizations(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if fullUser == nil {
		fullUser = user
	}

	tokens, err := s.issueAuthTokens(ctx, fullUser, token.Expiry)
	if err != nil {
		return nil, nil, err
	}

	return fullUser, tokens, nil
}

// HandleMemberOAuth2Callback behaves like HandleOAuth2Callback but additionally upserts the role.
func (s *GoogleAuthService) HandleMemberOAuth2Callback(
	ctx context.Context,
	authorizationResponse string,
	state string,
	redirectURI string,
) (*domain.User, *AuthTokens, error) {
	user, tokens, err := s.HandleOAuth2Callback(ctx, authorizationResponse, state, redirectURI)
	if err != nil {
		return nil, nil, err
	}

	queryVals, err := url.ParseQuery(state)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid state: %w", err)
	}

	orgID, _ := parseUUID(queryVals.Get("organization_id"))
	jobTitle := queryVals.Get("job_title")
	roleName := domain.RoleName(queryVals.Get("role"))
	if roleName == "" {
		roleName = domain.RoleNameMember
	}

	if orgID != uuid.Nil {
		role := &domain.Role{
			UserID:         user.ID,
			OrganizationID: orgID,
			Name:           roleName,
			Status:         domain.RoleStatusActive,
			JobTitle:       jobTitle,
		}
		if _, err := s.roleRepo.Upsert(ctx, role); err != nil {
			return nil, nil, err
		}
		fullUser, err := s.userRepo.FindWithOrganizations(ctx, user.ID)
		if err != nil {
			return nil, nil, err
		}
		if fullUser != nil {
			user = fullUser
		}
		_, expiry, tokenErr := extractTokenStore(user.Credentials)
		if tokenErr != nil {
			expiry = time.Time{}
		}
		tokens, err = s.issueAuthTokens(ctx, user, expiry)
		if err != nil {
			return nil, nil, err
		}
	}

	return user, tokens, nil
}

// HandleInviteMemberCallback processes invite callbacks by returning either an authorization URL or tokens.
func (s *GoogleAuthService) HandleInviteMemberCallback(
	ctx context.Context,
	state string,
	redirectURI string,
) (*InviteMemberResult, error) {
	if state == "" {
		return nil, errors.New("missing state parameter")
	}

	decodedState, err := url.QueryUnescape(state)
	if err != nil {
		return nil, fmt.Errorf("invalid state encoding: %w", err)
	}

	queryVals, err := url.ParseQuery(decodedState)
	if err != nil {
		return nil, fmt.Errorf("invalid state payload: %w", err)
	}

	email := queryVals.Get("email")
	if email == "" {
		return nil, errors.New("state payload missing email")
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// When user missing or lacks credentials -> return authorization URL
	if user == nil || len(user.Credentials) == 0 {
		url, err := s.GetAuthorizationURL(ctx, redirectURI, decodedState, email)
		if err != nil {
			return nil, err
		}
		return &InviteMemberResult{AuthorizationURL: url}, nil
	}

	orgID, _ := parseUUID(queryVals.Get("organization_id"))
	jobTitle := queryVals.Get("job_title")
	roleName := domain.RoleName(queryVals.Get("role"))
	if roleName == "" {
		roleName = domain.RoleNameMember
	}

	if orgID != uuid.Nil {
		role := &domain.Role{
			UserID:         user.ID,
			OrganizationID: orgID,
			Name:           roleName,
			Status:         domain.RoleStatusActive,
			JobTitle:       jobTitle,
		}
		if _, err := s.roleRepo.Upsert(ctx, role); err != nil {
			return nil, err
		}
	}

	fullUser, err := s.userRepo.FindWithOrganizations(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if fullUser != nil {
		user = fullUser
	}

	_, expiry, err := extractTokenStore(user.Credentials)
	if err != nil {
		return nil, err
	}

	tokens, err := s.issueAuthTokens(ctx, user, expiry)
	if err != nil {
		return nil, err
	}

	return &InviteMemberResult{Tokens: tokens, User: user, AuthorizationURL: ""}, nil
}

// RefreshAccessToken refreshes Google credentials and returns new auth tokens.
func (s *GoogleAuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*domain.User, *AuthTokens, error) {
	if s.oauthClient == nil {
		return nil, nil, errors.New("google oauth client not configured")
	}
	claims, _, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, errors.New("user not found")
	}

	tokenMap, _, err := extractTokenStore(user.Credentials)
	if err != nil {
		return nil, nil, err
	}
	refreshValue, _ := tokenMap["refresh_token"].(string)
	if refreshValue == "" {
		return nil, nil, errors.New("missing refresh token in stored credentials")
	}

	newToken, err := s.oauthClient.RefreshToken(ctx, refreshValue)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to refresh google token: %w", err)
	}

	tokenMap["token"] = newToken.AccessToken
	tokenMap["expiry"] = newToken.Expiry.UTC().Format(time.RFC3339)
	if newToken.RefreshToken != "" {
		tokenMap["refresh_token"] = newToken.RefreshToken
	}

	if err := user.UpdateGoogleTokenStore(tokenMap); err != nil {
		return nil, nil, err
	}
	if err := s.userRepo.UpdateByID(ctx, user.ID, map[string]interface{}{"credentials": user.Credentials}); err != nil {
		return nil, nil, err
	}

	fullUser, err := s.userRepo.FindWithOrganizations(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if fullUser != nil {
		user = fullUser
	}

	tokens, err := s.issueAuthTokens(ctx, user, newToken.Expiry)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Signout revokes Google tokens associated with the provided credentials map.
func (s *GoogleAuthService) Signout(ctx context.Context, credentials map[string]interface{}) error {
	if credentials == nil {
		return errors.New("credentials missing")
	}
	if s.oauthClient == nil {
		return errors.New("google oauth client not configured")
	}
	token, _ := credentials["token"].(string)
	if token == "" {
		return errors.New("no access token found in credentials")
	}
	return s.oauthClient.RevokeToken(ctx, token)
}

// Authorize validates the access token and fetches the corresponding user.
func (s *GoogleAuthService) Authorize(ctx context.Context, accessToken string) (*domain.User, error) {
	claims, _, err := s.jwtManager.ParseAccessToken(accessToken)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *GoogleAuthService) upsertGoogleUser(
	ctx context.Context,
	userInfo *googleoauth.UserInfo,
	token *googleoauth.Token,
	cfg *oauth2.Config,
) (*domain.User, error) {
	if userInfo == nil || userInfo.Email == "" {
		return nil, errors.New("user info missing email")
	}

	user, err := s.userRepo.FindByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, err
	}

	credentialsPayload := buildCredentialPayload(cfg, token)
	credBytes, err := json.Marshal(credentialsPayload)
	if err != nil {
		return nil, err
	}
	credJSON := domain.JSON(credBytes)

	if user == nil {
		newUser := &domain.User{
			Email:       userInfo.Email,
			Name:        userInfo.Name,
			Picture:     userInfo.Picture,
			Provider:    "google",
			Credentials: credJSON,
		}
		return s.userRepo.Create(ctx, newUser)
	}

	updateData := map[string]interface{}{
		"name":     userInfo.Name,
		"picture":  userInfo.Picture,
		"provider": "google",
	}
	if token.RefreshToken != "" {
		updateData["credentials"] = credJSON
	}

	if err := s.userRepo.UpdateByID(ctx, user.ID, updateData); err != nil {
		return nil, err
	}

	updatedUser, err := s.userRepo.FindByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if updatedUser != nil {
		user = updatedUser
	}
	return user, nil
}

func (s *GoogleAuthService) clientForRedirect(redirectURI string) (*googleoauth.Client, error) {
	cfg := s.oauthClient.GetConfig()
	if cfg == nil {
		return nil, errors.New("google oauth config unavailable")
	}
	if redirectURI == "" || cfg.RedirectURL == redirectURI {
		return s.oauthClient, nil
	}
	return googleoauth.NewClient(googleoauth.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  redirectURI,
	}), nil
}

func (s *GoogleAuthService) issueAuthTokens(ctx context.Context, user *domain.User, expiry time.Time) (*AuthTokens, error) {
	roleClaims := s.buildRoleClaims(ctx, user.ID)

	accessClaims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
		"roles": roleClaims,
	}
	refreshClaims := jwt.MapClaims{
		"sub": user.ID.String(),
	}
	idClaims := jwt.MapClaims{
		"sub":     user.ID.String(),
		"email":   user.Email,
		"name":    user.Name,
		"picture": user.Picture,
	}

	accessToken, err := s.jwtManager.CreateAccessToken(accessClaims, expiry)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.jwtManager.CreateRefreshToken(refreshClaims, time.Time{})
	if err != nil {
		return nil, err
	}
	idToken, err := s.jwtManager.CreateIDToken(idClaims, time.Time{})
	if err != nil {
		return nil, err
	}

	expiresIn := accessToken.Expiry.Sub(time.Now().UTC())
	if expiresIn < 0 {
		expiresIn = 0
	}

	return &AuthTokens{
		TokenType:    "Bearer",
		AccessToken:  accessToken.Value,
		RefreshToken: refreshToken.Value,
		IDToken:      idToken.Value,
		ExpiresIn:    expiresIn.Milliseconds(),
	}, nil
}

func buildRoleClaimsWithRelations(ctx context.Context, user *domain.User, relationsHelper *repositoryHelper.RelationsHelper) []map[string]interface{} {
	// Load user with roles using relations package
	userWithRoles, err := relationsHelper.GetUserWithRoles(ctx, user.ID)
	if err != nil || userWithRoles == nil {
		return []map[string]interface{}{}
	}

	claims := make([]map[string]interface{}, 0, len(userWithRoles.Roles))
	for _, role := range userWithRoles.Roles {
		orgID := role.OrganizationID.String()
		orgName := ""

		// Since role.Organization might not be directly available due to domain structure
		// we'll keep organization name empty for now, but preserve the structure
		claims = append(claims, map[string]interface{}{
			"name":            string(role.Name),
			"organization_id": orgID,
			"organization": map[string]interface{}{
				"name": orgName,
			},
		})
	}
	return claims
}

// Fallback method for backward compatibility
func (s *GoogleAuthService) buildRoleClaims(ctx context.Context, userID uuid.UUID) []map[string]interface{} {
	roles, err := s.roleRepo.FindByUserID(ctx, userID)
	if err != nil || len(roles) == 0 {
		return []map[string]interface{}{}
	}

	claims := make([]map[string]interface{}, 0, len(roles))
	for _, role := range roles {
		// Skip soft-deleted roles to match active memberships
		if role.IsDeleted {
			continue
		}
		orgID := role.OrganizationID.String()
		claims = append(claims, map[string]interface{}{
			"name":            string(role.Name),
			"organization_id": orgID,
			"organization": map[string]interface{}{
				"name": "",
			},
		})
	}
	return claims
}

func extractTokenStore(raw domain.JSON) (map[string]interface{}, time.Time, error) {
	if raw == nil {
		return nil, time.Time{}, errors.New("credentials empty")
	}
	data := map[string]interface{}{}
	if err := raw.Unmarshal(&data); err != nil {
		return nil, time.Time{}, err
	}
	store, err := domain.NewGoogleTokenStoreFromMap(data)
	if err != nil {
		return nil, time.Time{}, err
	}
	return store.ToMap(), store.Expiry, nil
}

func buildCredentialPayload(cfg *oauth2.Config, token *googleoauth.Token) map[string]interface{} {
	scopes := append([]string{}, cfg.Scopes...)
	expiry := token.Expiry.UTC().Format(time.RFC3339)
	payload := map[string]interface{}{
		"token":         token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_uri":     cfg.Endpoint.TokenURL,
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"scopes":        scopes,
		"expiry":        expiry,
		"token_type":    token.TokenType,
	}
	return payload
}

func extractQueryParam(rawURL, key string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsed.Query().Get(key), nil
}

func parseUUID(value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(value)
}
