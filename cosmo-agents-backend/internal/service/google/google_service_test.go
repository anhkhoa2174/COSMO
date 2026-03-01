package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	inboundRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	"golang.org/x/oauth2"
)

func newGoogleAdsService(t *testing.T) (*GoogleAdsService, context.Context, *campaignRepo.CampaignRepository, *inboundRepo.InboundLeadFormRepository, *contactRepo.ListContactRepository) {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.InboundLeadForm{},
		&domain.FormField{},
		&domain.InboundLeadFormListContactAssociation{},
		&domain.Campaign{},
	))
	// Minimal list_contacts table to avoid GIN index on contacts in SQLite
	require.NoError(t, db.Exec(`CREATE TABLE list_contacts (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		name TEXT,
		source TEXT,
		source_id TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		hubspot_id TEXT,
		organization_id TEXT,
		is_deleted BOOLEAN DEFAULT false
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE list_contact_association (
		id TEXT PRIMARY KEY,
		list_contact_id TEXT,
		contact_id TEXT
	);`).Error)

	inbound := inboundRepo.NewInboundLeadFormRepository(db)
	contact := contactRepo.NewContactRepository(db)
	list := contactRepo.NewListContactRepository(db)
	campaign := campaignRepo.NewCampaignRepository(db)

	service := NewGoogleAdsService(inbound, contact, list, campaign, nil, "http://localhost")
	return service, context.Background(), campaign, inbound, list
}

func TestGoogleAdsService_GenerateWebhook(t *testing.T) {
	svc, ctx, campaignRepo, _, listRepo := newGoogleAdsService(t)

	userID := uuid.New()
	campaignID := uuid.New()
	listID := uuid.New()

	require.NoError(t, campaignRepo.GetDB().Create(&domain.Campaign{
		Base:           domain.Base{ID: campaignID},
		UserID:         userID,
		Status:         domain.CampaignStatusActive,
		OrganizationID: nil,
	}).Error)
	require.NoError(t, listRepo.GetDB().Create(&domain.ListContact{
		Base:   domain.Base{ID: listID},
		UserID: userID,
		Name:   "List",
	}).Error)

	webhook, err := svc.GenerateWebhook(ctx, CreateWebhookRequest{
		CampaignID:    campaignID,
		ContactListID: listID,
		Name:          "Webhook",
		Slug:          "google-webhook",
		UserID:        userID,
	})
	require.NoError(t, err)
	assert.Contains(t, webhook, "google-webhook")

	// slug exists branch
	webhook2, err := svc.GenerateWebhook(ctx, CreateWebhookRequest{
		CampaignID:    campaignID,
		ContactListID: listID,
		Name:          "Webhook",
		Slug:          "google-webhook",
		UserID:        userID,
	})
	assert.Equal(t, ErrWebhookSlugExists, err)
	assert.Empty(t, webhook2)

	// campaign missing
	_, err = svc.GenerateWebhook(ctx, CreateWebhookRequest{
		CampaignID:    uuid.New(),
		ContactListID: listID,
		Name:          "Webhook",
		Slug:          "another",
		UserID:        userID,
	})
	assert.Equal(t, ErrCampaignNotFound, err)
}

func TestGoogleAdsService_ProcessWebhook_Errors(t *testing.T) {
	svc, ctx, campaignRepo, inboundRepo, _ := newGoogleAdsService(t)

	userID := uuid.New()
	campaignID := uuid.New()
	listID := uuid.New()

	require.NoError(t, campaignRepo.GetDB().Create(&domain.Campaign{
		Base:           domain.Base{ID: campaignID},
		UserID:         userID,
		Status:         domain.CampaignStatusPaused,
		OrganizationID: nil,
		AgentID:        func() *uuid.UUID { v := uuid.New(); return &v }(),
	}).Error)

	meta := map[string]string{
		"campaign_id":     campaignID.String(),
		"contact_list_id": listID.String(),
	}
	var metaJSON domain.JSONB
	require.NoError(t, metaJSON.Marshal(meta))
	require.NoError(t, inboundRepo.GetDB().Create(&domain.InboundLeadForm{
		Base:       domain.Base{ID: uuid.New()},
		Slug:       "slug",
		Name:       "Form",
		UIMetadata: metaJSON,
	}).Error)

	// Campaign inactive
	err := svc.ProcessWebhook(ctx, "slug", GoogleAdsLeadPayload{
		LeadID:         "1",
		UserColumnData: []GoogleAdsColumnEntry{{ColumnID: "EMAIL", StringValue: "lead@example.com"}},
	})
	assert.Equal(t, ErrCampaignInactive, err)

	// Missing agent
	require.NoError(t, campaignRepo.GetDB().Model(&domain.Campaign{}).Where("id = ?", campaignID).Update("status", domain.CampaignStatusActive).Error)
	require.NoError(t, campaignRepo.GetDB().Model(&domain.Campaign{}).Where("id = ?", campaignID).Update("agent_id", nil).Error)
	err = svc.ProcessWebhook(ctx, "slug", GoogleAdsLeadPayload{
		LeadID:         "1",
		UserColumnData: []GoogleAdsColumnEntry{{ColumnID: "EMAIL", StringValue: "lead@example.com"}},
	})
	assert.Equal(t, ErrMissingAgent, err)

	// Missing email => no error, early return
	require.NoError(t, campaignRepo.GetDB().Model(&domain.Campaign{}).Where("id = ?", campaignID).Update("agent_id", uuid.New()).Error)
	err = svc.ProcessWebhook(ctx, "slug", GoogleAdsLeadPayload{
		LeadID:         "1",
		UserColumnData: []GoogleAdsColumnEntry{{ColumnID: "FIRST_NAME", StringValue: "Lead"}},
	})
	assert.NoError(t, err)

	// Unknown slug => nil
	err = svc.ProcessWebhook(ctx, "unknown", GoogleAdsLeadPayload{})
	assert.NoError(t, err)
}

func TestGoogleAdsHelpers(t *testing.T) {
	data := mapLeadData(GoogleAdsLeadPayload{
		UserColumnData: []GoogleAdsColumnEntry{
			{ColumnID: "EMAIL", StringValue: "a@example.com"},
			{ColumnID: "UNKNOWN", StringValue: "skip"},
		},
	})
	assert.Equal(t, "a@example.com", data["email"])

	assert.Equal(t, domain.NOT_AVAILABLE, fallback(" "))
	assert.Equal(t, "value", fallback(" value "))
}

// Minimal GoogleAuthService error-path coverage
func TestGoogleAuthService_GetAuthURLAndCallbackErrors(t *testing.T) {
	svc := &GoogleAuthService{}
	_, err := svc.GetAuthorizationURL(context.Background(), "", "", "")
	assert.Error(t, err)

	_, _, err = svc.HandleOAuth2Callback(context.Background(), "", "", "")
	assert.Error(t, err)
}

func TestGoogleAuthService_IssueTokens(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Role{}))

	manager := auth.NewJWTManager("secret", time.Hour, "", 0, "")
	svc := &GoogleAuthService{
		jwtManager: manager,
		roleRepo:   roleRepo.NewRoleRepository(db),
	}

	tokens, err := svc.issueAuthTokens(context.Background(), &domain.User{
		Base:  domain.Base{ID: uuid.New()},
		Email: "user@example.com",
		Name:  "User",
	}, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.NotEmpty(t, tokens.IDToken)
}

type stubTransport struct {
	responder func(req *http.Request) *http.Response
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.responder == nil {
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	}
	return s.responder(req), nil
}

// Full OAuth flow happy-path with stubbed HTTP.
func TestGoogleAuthService_HandleOAuth2CallbackSuccess(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Role{}))

	transport := &stubTransport{
		responder: func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "token") {
				body := `{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
			}
			body := `{"email":"test@example.com","name":"Test User","picture":"pic"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
		},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})

	oauthClient := googleoauth.NewClient(googleoauth.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURI:  "http://localhost/callback",
	})

	userRepo := userRepo.NewUserRepository(db)
	roleRepo := roleRepo.NewRoleRepository(db)
	jwtManager := auth.NewJWTManager("secret", time.Hour, "refresh", 0, "id")

	svc := &GoogleAuthService{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		jwtManager:  jwtManager,
		oauthClient: oauthClient,
	}

	user, tokens, err := svc.HandleOAuth2Callback(ctx, "http://localhost/callback?code=abc", "", "")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestGoogleAuthService_InviteMemberAndRefreshAndAuthorize(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Role{}))

	transport := &stubTransport{
		responder: func(req *http.Request) *http.Response {
			body := `{"access_token":"new-acc","refresh_token":"new-ref","token_type":"Bearer","expires_in":3600}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
		},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})

	oauthClient := googleoauth.NewClient(googleoauth.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURI:  "http://localhost/callback",
	})

	userRepo := userRepo.NewUserRepository(db)
	roleRepo := roleRepo.NewRoleRepository(db)
	jwtManager := auth.NewJWTManager("secret", time.Hour, "refresh", 0, "id")

	svc := &GoogleAuthService{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		jwtManager:  jwtManager,
		oauthClient: oauthClient,
	}

	// Invite flow with missing user -> returns auth URL
	state := url.QueryEscape("email=new@example.com&organization_id=&role=&job_title=")
	invite, err := svc.HandleInviteMemberCallback(ctx, state, "http://localhost/callback")
	require.NoError(t, err)
	require.NotNil(t, invite)
	assert.NotEmpty(t, invite.AuthorizationURL)

	// Seed user with credentials for refresh/authorize
	tokenPayload := map[string]interface{}{
		"token":         "old",
		"refresh_token": "refresh-old",
		"token_uri":     oauthClient.GetConfig().Endpoint.TokenURL,
		"client_id":     "cid",
		"client_secret": "secret",
		"expiry":        time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"scopes":        []string{"scope"},
	}
	credBytes, _ := json.Marshal(tokenPayload)
	user := &domain.User{
		Base:        domain.Base{ID: uuid.New()},
		Email:       "new@example.com",
		Name:        "New User",
		Credentials: domain.JSON(credBytes),
	}
	_, err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	refreshToken, err := jwtManager.CreateRefreshToken(jwt.MapClaims{"sub": user.ID.String()}, time.Time{})
	require.NoError(t, err)

	refreshedUser, tokens, err := svc.RefreshAccessToken(ctx, refreshToken.Value)
	require.NoError(t, err)
	require.NotNil(t, refreshedUser)
	assert.Equal(t, user.ID, refreshedUser.ID)
	assert.Equal(t, "Bearer", tokens.TokenType)

	accessToken, err := jwtManager.CreateAccessToken(jwt.MapClaims{"sub": user.ID.String()}, time.Now().Add(time.Hour))
	require.NoError(t, err)
	authorized, err := svc.Authorize(ctx, accessToken.Value)
	require.NoError(t, err)
	assert.Equal(t, user.ID, authorized.ID)
}

func TestGoogleAuthService_SignoutAndHelpers(t *testing.T) {
	svc := &GoogleAuthService{}

	// credentials missing
	err := svc.Signout(context.Background(), nil)
	assert.Error(t, err)

	err = svc.Signout(context.Background(), map[string]interface{}{})
	assert.Error(t, err)

	// parse helpers
	val, err := extractQueryParam("http://example.com?code=123", "code")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)

	id, err := parseUUID("")
	assert.NoError(t, err)
	assert.Equal(t, uuid.Nil, id)
}

func TestGoogleAuthService_SignoutSuccessAndHelpers(t *testing.T) {
	transport := &stubTransport{
		responder: func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}
		},
	}
	oldDefault := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: transport}
	defer func() { http.DefaultClient = oldDefault }()

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, http.DefaultClient)
	oauthClient := googleoauth.NewClient(googleoauth.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURI:  "http://cb",
	})

	svc := &GoogleAuthService{oauthClient: oauthClient}
	err := svc.Signout(ctx, map[string]interface{}{"token": "abc"})
	assert.NoError(t, err)

	// non-200 should error; override default client
	transport.responder = func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(``)), Header: make(http.Header)}
	}
	err = svc.Signout(ctx, map[string]interface{}{"token": "abc"})
	assert.Error(t, err)
}

func TestGoogleAuthService_TokenHelpers(t *testing.T) {
	cfg := &oauth2.Config{ClientID: "cid", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: "http://token"}, Scopes: []string{"scope1"}}
	token := &googleoauth.Token{AccessToken: "acc", RefreshToken: "ref", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	payload := buildCredentialPayload(cfg, token)
	assert.Equal(t, "acc", payload["token"])
	assert.Equal(t, "ref", payload["refresh_token"])

	// invalid JSON in extractTokenStore
	_, _, err := extractTokenStore(domain.JSON(nil))
	assert.Error(t, err)
}

func TestGoogleAuthService_ClientForRedirect(t *testing.T) {
	oauthClient := googleoauth.NewClient(googleoauth.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURI:  "http://default",
	})
	svc := &GoogleAuthService{oauthClient: oauthClient}
	client, err := svc.clientForRedirect("http://override")
	assert.NoError(t, err)
	assert.NotEqual(t, oauthClient, client)
}
