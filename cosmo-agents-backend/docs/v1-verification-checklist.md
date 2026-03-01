# V1 API Verification Checklist

Date: 2025-10-20

This checklist tracks verification status for all V1 modules and their endpoints. It is based on current routes defined in `cmd/server/routes_v1.go`.

## Legend
- [x] = Completed/Verified
- [ ] = Pending

Verification items per module:
- [ ] API parity verified (spec vs. implementation)
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated (README/API docs)

---

## Status snapshot

Reviewed via recent PRs:
- [x] auth (PR #8 parity, PR #10 JWT expiry fixes)
- [x] gmail (PR #6 v1 parity)
- [x] knowledge (PR #4 align with Python)

Pending verify:
- [x] agent
- [ ] ai (companies, emails)
- [ ] campaign
- [ ] contact
- [ ] conversation
- [ ] custom_field
- [ ] email
- [ ] file
- [ ] google_ads
- [ ] hubspot
- [ ] inbound_lead_form
- [ ] lab
- [ ] lead_form_integration
- [ ] list_contact
- [ ] meta
- [ ] organization
- [ ] outlook
- [ ] pubsub
- [ ] sale_rep
- [ ] task
- [ ] task_enqueue
- [ ] template
- [ ] user
- [ ] workflow

Note: Test files currently present for some handlers: agent, auth, campaign, contact, email, organization, template, user (based on `internal/handler/v1/*_test.go`).

---

## Module checklists

### Auth
Endpoints:
- POST /v1/auth/login
- POST /v1/auth/refresh
- GET  /v1/auth/me
- POST /v1/auth/logout

Checks:
- [x] API parity verified
- [x] AuthN/AuthZ behavior verified
- [x] Validation and error handling verified
- [x] Happy path manual check
- [x] Integration/handler tests present
- [x] Docs updated

Notes: Verified via PRs #8 and #10.

---

### User
Endpoints:
- GET  /v1/user/:id
- GET  /v1/user/:id/organizations
- GET  /v1/user
- POST /v1/users/:user_id/personal-api-keys
- GET  /v1/users/:user_id/personal-api-keys
- DELETE /v1/users/:user_id/personal-api-keys/:key_id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for user)
- [ ] Docs updated

Notes:
- Confirm org scoping and permissions for personal API keys.

---

### Organization
Endpoints:
- POST /v1/organizations
- GET  /v1/organization/:id
- GET  /v1/organization
- PATCH /v1/organizations/:id
- POST /v1/organizations/:id/members
- POST /v1/organizations/assign

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for organization)
- [ ] Docs updated

Notes:
- Different singular/plural paths (organization vs organizations).

---

### Contact
Endpoints:
- POST   /v1/contacts
- GET    /v1/contact/:id
- GET    /v1/contact
- GET    /v1/contacts/values
- PATCH  /v1/contacts/:id
- DELETE /v1/contacts
- POST   /v1/contacts/import-csv
- POST   /v1/contacts/import-hubspot

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for contact)
- [ ] Docs updated

---

### Campaign
Endpoints:
- POST   /v1/campaigns
- POST   /v1/campaigns/search
- GET    /v1/campaign/:id
- GET    /v1/campaign/:id/relations
- GET    /v1/campaign
- PATCH  /v1/campaigns/:id
- PATCH  /v1/campaigns/:id/client-metadata
- DELETE /v1/campaigns/:id
- POST   /v1/campaigns/:id/generate
- PATCH  /v1/campaigns/:id/save-outreach
- POST   /v1/campaigns/:id/assign
- POST   /v1/campaigns/:id/follow-up-schedule
- DELETE /v1/campaigns/:id/notifications

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for campaign)
- [ ] Docs updated

---

### Agent
Endpoints:
- POST /v1/agents
- GET  /v1/agents/:id
- GET  /v1/agents
- PUT  /v1/agents/:id
- DELETE /v1/agents/:id

Checks:
- [x] API parity verified
- [x] AuthN/AuthZ behavior verified
- [x] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for agent)
- [ ] Docs updated

---

### Task
Endpoints:
- POST /v1/task
- GET  /v1/task/:id
- GET  /v1/task
- PUT  /v1/task/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Template
Endpoints:
- POST   /v1/template
- GET    /v1/template/:id
- GET    /v1/template
- PUT    /v1/template/:id
- DELETE /v1/template/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for template)
- [ ] Docs updated

---

### Email
Endpoints:
- GET /v1/email/:id
- GET /v1/email
- PUT /v1/email/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present (some tests exist for email)
- [ ] Docs updated

---

### Conversation
Endpoints:
- POST /v1/conversation
- GET  /v1/conversation/:id
- GET  /v1/conversation
- PUT  /v1/conversation/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Gmail
Endpoints:
- POST /v1/gmail/auth/url
- GET  /v1/gmail/auth/callback
- POST /v1/gmail/auth/refresh
- GET  /v1/gmail/profile/:agent_id
- POST /v1/gmail/send

Checks:
- [x] API parity verified
- [x] AuthN/AuthZ behavior verified
- [x] Validation and error handling verified
- [x] Happy path manual check
- [ ] Integration/handler tests present
- [x] Docs updated

Notes: Verified via PR #6.

---

### Task Enqueue
Endpoints:
- POST /v1/tasks/enqueue/send-email
- POST /v1/tasks/enqueue/execute-campaign
- POST /v1/tasks/enqueue/schedule-tasks
- POST /v1/tasks/enqueue/sync-agent

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Sale Rep
Endpoints:
- POST   /v1/sale-reps
- GET    /v1/sale-reps/:id
- GET    /v1/sale-reps
- POST   /v1/sale-reps/search
- PATCH  /v1/sale-reps/:id
- DELETE /v1/sale-reps/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Custom Field
Endpoints:
- POST   /v1/custom-fields
- GET    /v1/custom-fields
- PATCH  /v1/custom-fields/:id
- DELETE /v1/custom-fields/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### List Contact
Endpoints:
- POST   /v1/list-contacts
- GET    /v1/list-contacts/:id
- POST   /v1/list-contacts/search
- PATCH  /v1/list-contacts/:id
- DELETE /v1/list-contacts

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Workflow
Endpoints:
- POST   /v1/workflows
- GET    /v1/workflows/:id
- GET    /v1/workflows
- PATCH  /v1/workflows/:id
- DELETE /v1/workflows/:id

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Knowledge
Endpoints:
- POST   /v1/knowledge
- GET    /v1/knowledge/:id
- GET    /v1/knowledge
- POST   /v1/knowledge/search
- PATCH  /v1/knowledge/:id
- DELETE /v1/knowledge/:id

Checks:
- [x] API parity verified
- [x] AuthN/AuthZ behavior verified
- [x] Validation and error handling verified
- [x] Happy path manual check
- [ ] Integration/handler tests present
- [x] Docs updated

Notes: Verified via PR #4.

---

### AI
Endpoints:
- GET  /v1/ai/companies/extract
- POST /v1/ai/emails/generate
- POST /v1/ai/emails/classify-intent
- POST /v1/ai/emails/reply

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### HubSpot
Endpoints:
- GET /v1/hubspot/authorize
- GET /v1/hubspot/callback
- GET /v1/hubspot/users/me

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Outlook
Endpoints:
- GET  /v1/outlook/authorize
- GET  /v1/outlook/oauth2callback
- GET  /v1/outlook/refresh_token
- GET  /v1/outlook/contacts
- GET  /v1/outlook/emails
- POST /v1/outlook/send_email
- POST /v1/outlook/reply_email
- POST /v1/outlook/forward_email

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Google Ads
Endpoints:
- POST /v1/google-ads/generate-webhook
- POST /v1/google-ads/:slug/webhook

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified (webhook security)
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Inbound Lead Forms
Endpoints:
- POST   /v1/inbound-lead-forms
- GET    /v1/inbound-lead-forms
- GET    /v1/inbound-lead-forms/:identifier
- PUT    /v1/inbound-lead-forms/:identifier
- DELETE /v1/inbound-lead-forms/:identifier
- POST   /v1/inbound-lead-forms/:identifier/submit

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Lead Form Integrations
Endpoints:
- POST /v1/lead-form-integrations
- GET  /v1/lead-form-integrations/:id
- DELETE /v1/lead-form-integrations/:id
- PUT /v1/lead-form-integrations/:id/field-mappings

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### PubSub
Endpoints:
- POST /v1/pubsub/publish
- GET  /v1/pubsub/subscribe/:topic
- GET  /v1/pubsub/topic/:topic

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified (topic access control)
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Files (S3)
Endpoints (enabled only if file handler is configured):
- GET  /v1/files
- POST /v1/files/search
- POST /v1/files/s3
- GET  /v1/files/s3
- GET  /v1/files/s3/list
- GET  /v1/files/s3/presigned-url

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Lab
Endpoints:
- POST /v1/lab/generate-sample-response
- GET  /v1/lab/conversations/:conversation_id/emails

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### Meta (Facebook)
Endpoints:
- GET  /v1/meta/oauth2/auth-url
- GET  /v1/meta/oauth2/callback
- GET  /v1/meta/webhook (verify)
- POST /v1/meta/subscribe
- POST /v1/meta/webhook (receive lead notification)

Checks:
- [ ] API parity verified
- [ ] AuthN/AuthZ behavior verified (webhook security/verification)
- [ ] Validation and error handling verified
- [ ] Happy path manual check
- [ ] Integration/handler tests present
- [ ] Docs updated

---

### List of open items
Use this section to capture cross-cutting tasks:
- [ ] Confirm test coverage strategy for modules without handler tests
- [ ] Ensure Swagger/OpenAPI docs sync with routes
- [ ] Verify middleware behavior for `EnableAuth` toggling
- [ ] Add postman/insomnia collection references

---

Maintained by: Engineering
