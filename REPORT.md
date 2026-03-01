# COSMO Report (Draft)

**COVER PAGE**  
(Fill in based on the template in `main.pdf`)
- Project title: COSMO – [exact title]
- Advisor: [name]
- Students: [names]
- Student IDs: [IDs]
- Time/Location: [e.g., HCMC, 01/2026]

---

## 1. Introduction

### 1.1 System Description
COSMO is a campaign management system with core features: user authentication, user and organization management, contact management, contact lists, campaigns, agents, and task enqueueing. The system supports a clear flow from creating contacts → grouping into lists → creating campaigns → assigning agents → enqueueing tasks.

### 1.2 Problem Statement
Sales/marketing teams often struggle with scattered contact data and lack of a structured campaign workflow. COSMO addresses this by providing a centralized platform to manage contacts, campaigns, and task execution.

### 1.3 Challenges
- Implementing authentication and basic authorization.
- Organizing large contact datasets with efficient search.
- Synchronizing campaign–agent–task data.
- Keeping backend APIs and frontend UI aligned.

### 1.4 Reasons for Implementation
- Standardize contact and campaign workflows.
- Provide a clear operational view of data and status.
- Establish a foundation for future expansion.

### 1.5 Scope of Implementation
This project focuses only on the following modules:
- Authentication & Authorization
- User
- Organization
- Contact + Custom Fields
- Contact Lists
- Campaign
- Agent
- Task / Task Enqueue
- Inbound Lead Forms (supporting Contact Lists/Campaigns)

---

## 2. System Overview

### 2.1 Survey of Related Systems
Many CRM/marketing automation platforms exist but are complex and costly for a specialized project scope. COSMO focuses on a streamlined, easy‑to‑deploy core system.

### 2.2 Objectives
- Efficient user and organization management.
- Fast contact storage and retrieval.
- Campaign management based on contact lists.
- Agent assignment and task orchestration.
- Clear and maintainable system flow.

### 2.3 Stakeholders
- Project team
- Advisor
- Target users (sales/marketing staff)

### 2.4 Users
- **Regular user**: manage contacts and campaigns.
- **Admin** (if available): manage organization data.

---

## 3. System Requirements

### 3.1 Functional Requirements
- Auth: login, refresh, logout, get current user.
- User: fetch/update/delete current user.
- Organization: create, update, list organizations.
- Contact: CRUD, search, filter.
- Custom Fields: CRUD.
- Contact List: CRUD, search.
- Campaign: CRUD, search, assign agent, update status.
- Agent: CRUD, search.
- Task: create/update status.
- Task Enqueue: enqueue tasks (send email, sync agent, schedule tasks).
- Inbound Lead Forms: CRUD + submit.

### 3.2 Non‑Functional Requirements
- **Responsiveness:** Next.js UI with efficient rendering.
- **Reliability:** consistent API responses and auth middleware.
- **Maintainability:** modular domain separation.
- **Security:** JWT, refresh token, auth middleware.

---

## 4. System Analysis

### 4.1 Use Case Diagrams
- Authentication & Authorization
- User Management
- Organization Management
- Contact Management
- Campaign Management
- Agent Management
- Task Management

### 4.2 Use Case Scenarios
Examples:
- UC01: User login
- UC02: Create Contact
- UC03: Create Contact List
- UC04: Create Campaign
- UC05: Assign Agent to Campaign
- UC06: Enqueue Task

### 4.3 Activity Diagrams
Main flow: create contact → add to list → create campaign → assign agent → enqueue task.

### 4.4 Sequence Diagrams
- Login flow
- Campaign creation flow
- Task enqueue flow

### 4.5 Class Diagram
Key entities: User, Organization, Contact, ContactList, Campaign, Agent, Task, CustomField, InboundLeadForm.

---

## 5. System Architecture

### 5.1 Architectural Style
Client–server architecture.
Frontend: Next.js (React).
Backend: Go + Fiber.
Database: PostgreSQL.
Cache/queue: Redis.

### 5.2 Chosen Architecture
Layered architecture:
- Presentation (UI)
- API Controllers
- Services/Use Cases
- Repositories
- Database

### 5.3 Architecture Design
Backend and frontend are separated. Backend provides REST APIs, frontend consumes via HTTP clients.

### 5.4 Data Flow Diagram
Main flow:
User → UI → API → Database → API → UI
Auth and Task Enqueue modules control access and orchestration.

### 5.5 Deployment Diagram
- FE: Next.js app
- BE: Go service
- DB: PostgreSQL
- Redis

### 5.6 Technologies Used
**Frontend:** Next.js 14, React 18, TypeScript, Tailwind, React Query
**Backend:** Go 1.25, Fiber, JWT, GORM
**Database:** PostgreSQL
**Cache/Queue:** Redis

---

## 6. System Design

### 6.1 Module Design
- **Auth Module:** login/refresh/logout, JWT middleware.
- **User Module:** CRUD for current user.
- **Organization Module:** CRUD for organizations.
- **Contact Module:** CRUD + search.
- **Custom Field Module:** CRUD.
- **Contact List Module:** CRUD + search.
- **Campaign Module:** CRUD + assign agent.
- **Agent Module:** CRUD + search.
- **Task Module:** create/update.
- **Task Enqueue Module:** enqueue actions.

### 6.2 Database Design
Main tables: users, organizations, contacts, contact_lists, campaigns, agents, tasks, custom_fields, inbound_lead_forms, and join tables.

### 6.3 API Design
- `/v1/auth/*`
- `/v1/users/*`
- `/v1/organizations/*`
- `/v1/contacts/*`
- `/v1/custom-fields/*`
- `/v1/list-contacts/*`
- `/v1/campaigns/*`
- `/v1/agents/*`
- `/v1/task/*`
- `/v1/tasks/enqueue/*`
- `/v1/inbound-lead-forms/*`

### 6.4 UI/UX Design
Main screens:
- Login
- Campaigns
- Campaign Detail (simplified)
- Contacts
- Contact Lists
- Agents
- Organization Settings

---

## 7. System Implementation

- Backend implemented with Go Fiber, separated into handler/service/repo layers.
- Frontend built with Next.js App Router and React Query for API calls.
- JWT-based authentication and protected routes.
- Core APIs implemented within the defined scope.

---

## 8. Conclusion

### 8.1 Achieved Results
A functional, streamlined campaign management system with core modules implemented.

### 8.2 Strengths
- Simple and clear core flow.
- Easy to deploy and extend.
- Consistent API and UI integration.

### 8.3 Limitations
- Advanced features (AI, integrations, outreach workflows) are not included.
- Reporting/analytics features are not implemented.
