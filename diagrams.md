# COSMO System Diagrams

Paste từng block Mermaid vào https://mermaid.live để render thành hình.

---

## 1. System Architecture Overview (Kiến trúc tổng quan)

Diagram này thể hiện kiến trúc client-server tổng thể của COSMO: frontend giao tiếp với backend qua REST API, backend xử lý business logic và đẩy task vào hàng đợi cho background workers thực thi.

```mermaid
graph TB
    subgraph Client ["Client Layer"]
        Browser["Web Browser"]
    end

    subgraph Frontend ["Frontend — Next.js 14 / React 18"]
        UI["React UI Components"]
        AppRouter["Next.js App Router<br/>(Pages & Layouts)"]
        APIRoutes["Next.js API Routes<br/>(Server-side proxy)"]
        RQ["TanStack React Query<br/>(Server State Cache)"]
        KyClient["Ky HTTP Client<br/>(JWT auto-attach)"]
        AuthMW["Auth Middleware<br/>(JWT validation)"]
    end

    subgraph Backend ["Backend — Go / Fiber v3"]
        FiberApp["Fiber v3 HTTP Server"]
        MW["Middleware Stack<br/>CORS · JWT Auth · Rate Limit<br/>Request ID · Logging · Metrics"]
        Handlers["REST Handlers<br/>(V1 API Controllers)"]
        Services["Service Layer<br/>(Business Logic)"]
        Repos["Repository Layer<br/>(Data Access — GORM)"]
    end

    subgraph Workers ["Background Workers"]
        Asynq["Asynq Worker<br/>(Redis Task Queue)"]
    end

    subgraph DataStores ["Data Stores"]
        PG[("PostgreSQL<br/>Primary Database")]
        Redis[("Redis<br/>Cache & Task Queue")]
    end

    subgraph External ["External Services"]
        Google["Google OAuth"]
        Gmail["Gmail API"]
        OpenAI["OpenAI API<br/>(GPT-4)"]
        HubSpot["HubSpot CRM"]
    end

    Browser --> AuthMW
    AuthMW --> AppRouter
    AppRouter --> UI
    UI --> RQ
    RQ --> KyClient
    KyClient -->|"REST API + JWT"| FiberApp
    APIRoutes -->|"Server-side calls"| FiberApp

    FiberApp --> MW --> Handlers
    Handlers --> Services
    Services --> Repos
    Repos --> PG

    Services -->|"Enqueue tasks"| Redis
    Asynq -->|"Poll tasks"| Redis
    Asynq --> Services

    Services --> OpenAI
    Services --> Gmail
    Services --> HubSpot
    Handlers --> Google

    Asynq --> PG
```

**Mô tả:**
- **Client Layer:** Người dùng truy cập hệ thống qua trình duyệt web.
- **Frontend (Next.js 14):** Ứng dụng React sử dụng App Router để định tuyến, TanStack React Query để quản lý server state, và Ky HTTP client tự động gắn JWT token vào mỗi request. Auth Middleware kiểm tra JWT trước khi cho phép truy cập protected routes.
- **Backend (Go/Fiber v3):** REST API server theo kiến trúc phân lớp: Middleware → Handler → Service → Repository → Database. Middleware stack xử lý CORS, xác thực JWT, rate limiting, logging, và metrics.
- **Background Workers:** Asynq worker xử lý async tasks (gửi email, campaign execution, sync agent) từ Redis queue.
- **Data Stores:** PostgreSQL lưu trữ dữ liệu chính, Redis phục vụ caching và task queue.
- **External Services:** Tích hợp Google OAuth (xác thực), Gmail API (gửi/nhận email), OpenAI (sinh nội dung AI), HubSpot (đồng bộ CRM).

---

## 2. Backend Layered Architecture (Kiến trúc phân lớp Backend)

Diagram này thể hiện chi tiết kiến trúc phân lớp bên trong backend, từ HTTP request đến database, theo nguyên tắc separation of concerns.

```mermaid
graph TB
    subgraph HTTP ["HTTP Layer"]
        Request["HTTP Request"]
        MW1["Recovery Middleware"]
        MW2["Request ID"]
        MW3["Logger"]
        MW4["Prometheus Metrics"]
        MW5["CORS"]
        MW6["JWT Auth Middleware"]
        MW7["Rate Limiter"]
    end

    subgraph HandlerLayer ["Handler Layer (Controllers)"]
        AuthH["Auth Handler"]
        UserH["User Handler"]
        OrgH["Organization Handler"]
        ContactH["Contact Handler"]
        CampaignH["Campaign Handler"]
        AgentH["Agent Handler"]
        TaskH["Task Handler"]
        TemplateH["Template Handler"]
        CustomFieldH["Custom Field Handler"]
        ListContactH["List Contact Handler"]
        LeadFormH["Inbound Lead Form Handler"]
        TaskEnqueueH["Task Enqueue Handler"]
        EmailH["Email Handler"]
    end

    subgraph ServiceLayer ["Service Layer (Business Logic)"]
        AuthS["Auth Service"]
        AIEmailS["AI Email Service"]
        AICompanyS["AI Company Service"]
        GmailS["Gmail Service"]
        OutlookS["Outlook Service"]
        HubSpotS["HubSpot Service"]
        IntelligenceS["Intelligence Service"]
        KnowledgeS["Knowledge Service"]
        OrgS["Organization Service"]
        ScraperS["Scraper Service"]
        S3S["S3 Service"]
    end

    subgraph RepoLayer ["Repository Layer (Data Access)"]
        UserR["User Repo"]
        OrgR["Organization Repo"]
        ContactR["Contact Repo"]
        CampaignR["Campaign Repo"]
        AgentR["Agent Repo"]
        TaskR["Task Repo"]
        TemplateR["Template Repo"]
        EmailR["Email Repo"]
        OtherR["+ 30 more repos"]
    end

    subgraph DataLayer ["Data Layer"]
        GORM["GORM ORM"]
        PG[("PostgreSQL")]
    end

    Request --> MW1 --> MW2 --> MW3 --> MW4 --> MW5 --> MW6 --> MW7

    MW7 --> AuthH & UserH & OrgH & ContactH & CampaignH & AgentH & TaskH

    AuthH & UserH & OrgH --> AuthS & OrgS
    ContactH & CampaignH --> AIEmailS & AICompanyS & IntelligenceS
    AgentH --> GmailS & OutlookS
    TaskH & TaskEnqueueH --> AIEmailS

    AuthS & OrgS --> UserR & OrgR
    AIEmailS & AICompanyS --> ContactR & CampaignR & TemplateR
    GmailS & OutlookS --> AgentR & EmailR
    IntelligenceS --> ContactR

    UserR & OrgR & ContactR & CampaignR & AgentR & TaskR --> GORM --> PG
```

**Mô tả:**
- **HTTP Layer:** Mỗi request đi qua chuỗi middleware theo thứ tự: Recovery (xử lý panic) → Request ID (gán ID tracking) → Logger (ghi log) → Prometheus (thu thập metrics) → CORS → JWT Auth (xác thực token) → Rate Limiter (giới hạn request/IP).
- **Handler Layer:** 13 handler groups tương ứng với các module chức năng. Mỗi handler parse request, gọi service, và trả response.
- **Service Layer:** Chứa business logic. Các service có thể gọi lẫn nhau và tương tác với external APIs (OpenAI, Gmail, HubSpot).
- **Repository Layer:** 38 repositories cung cấp data access qua GORM ORM, tách biệt hoàn toàn khỏi business logic.
- **Data Layer:** GORM ORM translate Go structs sang SQL queries cho PostgreSQL.

---

## 3. Deployment Diagram (Sơ đồ triển khai)

Diagram này thể hiện cách các thành phần được triển khai trên các node vật lý/container.

```mermaid
graph TB
    subgraph UserDevice ["User Device"]
        Browser["Web Browser"]
    end

    subgraph AppServer ["Application Server"]
        subgraph FE ["Frontend Container"]
            NextJS["Next.js 14<br/>Port: 3000"]
        end
        subgraph BE ["Backend Container"]
            GoServer["Go Fiber Server<br/>Port: 8080"]
        end
        subgraph WK ["Worker Container"]
            AsynqWorker["Asynq Worker Process"]
        end
    end

    subgraph DBServer ["Database Server"]
        PostgreSQL["PostgreSQL<br/>Port: 5434"]
    end

    subgraph CacheServer ["Cache Server"]
        RedisServer["Redis<br/>Port: 6381"]
    end

    subgraph ExternalAPIs ["External APIs"]
        GoogleOAuth["Google OAuth 2.0"]
        GmailAPI["Gmail API"]
        OpenAIAPI["OpenAI API"]
        HubSpotAPI["HubSpot API"]
        AWSS3["AWS S3"]
    end

    Browser -->|"HTTPS"| NextJS
    NextJS -->|"HTTP REST"| GoServer
    GoServer -->|"TCP"| PostgreSQL
    GoServer -->|"TCP"| RedisServer
    AsynqWorker -->|"TCP"| RedisServer
    AsynqWorker -->|"TCP"| PostgreSQL
    GoServer -->|"HTTPS"| GoogleOAuth
    AsynqWorker -->|"HTTPS"| GmailAPI
    AsynqWorker -->|"HTTPS"| OpenAIAPI
    AsynqWorker -->|"HTTPS"| HubSpotAPI
    AsynqWorker -->|"HTTPS"| AWSS3
```

**Mô tả:**
- Hệ thống triển khai trên 3 container chính: Frontend (Next.js), Backend API (Go Fiber), và Asynq Worker.
- Frontend chạy trên port 3000, Backend trên port 8080. PostgreSQL dùng port 5434, Redis dùng port 6381 (custom ports để tránh conflict).
- Workers kết nối trực tiếp tới Redis (nhận task) và PostgreSQL (đọc/ghi dữ liệu), đồng thời gọi external APIs để thực thi task (gửi email qua Gmail, sinh nội dung qua OpenAI, đồng bộ CRM qua HubSpot, lưu file qua AWS S3).
- Docker Compose quản lý PostgreSQL và Redis trong môi trường development.

---

## 4a. Automated Email Campaign Flow (Luồng gửi email tự động)

Diagram này thể hiện luồng **tự động**: hệ thống gửi email hàng loạt qua Gmail API và tự động follow-up theo cadence đã cấu hình. Người dùng chỉ cần setup và bấm execute — mọi thứ sau đó do background worker xử lý.

```mermaid
sequenceDiagram
    actor User as BD Rep
    participant FE as Frontend<br/>(Next.js)
    participant API as Backend API<br/>(Go Fiber)
    participant DB as PostgreSQL
    participant Queue as Redis Queue
    participant Worker as Asynq Worker
    participant AI as OpenAI API
    participant Email as Gmail API

    Note over User,Email: Phase 1 — Setup Campaign
    User->>FE: Create contacts & contact list
    FE->>API: POST /v1/contacts<br/>POST /v1/list-contacts
    API->>DB: INSERT contacts, list_contacts

    User->>FE: Create campaign + assign list
    FE->>API: POST /v1/campaigns
    API->>DB: INSERT campaign

    User->>FE: Assign agent to campaign
    FE->>API: POST /v1/campaigns/:id/assign
    API->>DB: UPDATE campaign.agent_id

    Note over User,Email: Phase 2 — Generate Email Content
    User->>FE: Generate email templates
    FE->>API: POST /v1/campaigns/:id/generate
    API->>AI: Generate personalized emails<br/>(contact context + campaign strategy)
    AI-->>API: Email drafts
    API->>DB: INSERT templates
    API-->>FE: Generated templates

    Note over User,Email: Phase 3 — Execute Campaign
    User->>FE: Start campaign execution
    FE->>API: POST /v1/tasks/enqueue/execute-campaign
    API->>Queue: Enqueue campaign:execute task
    API-->>FE: Task enqueued ✓

    Worker->>Queue: Poll & dequeue task
    Worker->>DB: Load campaign + contacts + templates

    loop For each contact in campaign
        Worker->>Queue: Enqueue email:send task
        Worker->>DB: Load contact + template
        Worker->>Email: Send email via Gmail API
        Email-->>Worker: Send result
        Worker->>DB: INSERT email record<br/>UPDATE task status
    end

    Note over User,Email: Phase 4 — Follow-up
    Worker->>Queue: Enqueue schedule-tasks<br/>(follow-up cadence)
    Worker->>DB: UPDATE contact outreach_stage<br/>(COLD → NO_REPLY)
```

**Mô tả:**
Luồng gửi email tự động gồm 4 phase:
1. **Setup:** BD rep tạo contacts, gom vào contact list, tạo campaign, và gán agent (có Gmail credentials).
2. **Generate:** Hệ thống gọi OpenAI API để sinh email templates cá nhân hóa dựa trên contact context và campaign strategy.
3. **Execute:** Khi BD rep bấm execute, hệ thống enqueue task vào Redis. Asynq worker poll task, load dữ liệu từ DB, và gửi email qua Gmail API cho từng contact. Mỗi lần gửi đều ghi log email record và cập nhật task status.
4. **Follow-up:** Worker tự động schedule follow-up tasks theo cadence đã cấu hình và cập nhật outreach stage của contact.

---

## 4b. Manual Outreach Flow (Luồng outreach thủ công có AI hỗ trợ)

Diagram này thể hiện luồng **thủ công**: AI gợi ý contact cần liên hệ và sinh draft message, BD rep tự gửi qua LinkedIn/email thủ công, rồi log kết quả lại. Hệ thống theo dõi trạng thái outreach và tự động đề xuất bước tiếp theo.

```mermaid
sequenceDiagram
    actor User as BD Rep
    participant FE as Frontend<br/>(Next.js)
    participant API as Backend API<br/>(Go Fiber)
    participant DB as PostgreSQL
    participant AI as OpenAI API
    participant Worker as Asynq Worker
    participant External as LinkedIn / Email<br/>(Manual)

    Note over User,External: Step 1 — AI Suggests Contacts to Reach Out
    User->>FE: Open outreach dashboard
    FE->>API: GET suggested contacts<br/>(cold / follow-up / mixed)
    API->>DB: Query contacts by outreach state<br/>(days since last interaction, follow-up count)
    DB-->>API: Prioritized contact list
    API-->>FE: Display suggested contacts<br/>with next step (SEND, FOLLOW_UP_1, SET_MEETING...)

    Note over User,External: Step 2 — AI Generates Draft Message
    User->>FE: Select contact → Request draft
    FE->>API: Generate outreach draft<br/>(scenario + language)
    API->>DB: Load contact profile,<br/>interaction history, segment fit
    API->>AI: Generate personalized message<br/>(scenario: role-based / no-reply follow-up /<br/>post-reply / meeting confirmation...)
    AI-->>API: Draft message
    API-->>FE: Display draft for review

    Note over User,External: Step 3 — BD Rep Sends Manually
    User->>FE: Copy/edit draft
    User->>External: Send message via LinkedIn /<br/>personal email / phone call

    Note over User,External: Step 4 — Log Interaction
    User->>FE: Log interaction result
    FE->>API: Record interaction<br/>(channel, direction, sentiment)
    API->>DB: INSERT interaction log
    API->>DB: UPDATE outreach state<br/>(e.g., COLD → NO_REPLY,<br/>NO_REPLY → REPLIED)

    Note over User,External: Step 5 — Record Feedback on AI Draft
    User->>FE: Rate AI suggestion<br/>(used_draft / modified / wrote_own / skipped)
    FE->>API: Record feedback
    API->>DB: INSERT user feedback

    Note over User,External: Step 6 — Background State Recalculation
    Worker->>DB: Periodic: recalculate outreach states<br/>for contacts in WAIT state
    Worker->>DB: UPDATE next_step, follow_up_count<br/>when state transitions occur
```

**Mô tả:**
Luồng outreach thủ công gồm 6 bước:
1. **AI Suggest:** Hệ thống gợi ý danh sách contact cần liên hệ, ưu tiên theo trạng thái outreach (cold contacts, contacts cần follow-up) và thời gian kể từ lần tương tác cuối.
2. **AI Draft:** BD rep chọn contact, hệ thống gọi OpenAI sinh draft message cá nhân hóa theo scenario (first outreach, no-reply follow-up, post-reply, meeting confirmation...) và ngôn ngữ (Vietnamese/English).
3. **Manual Send:** BD rep copy/chỉnh sửa draft rồi tự gửi qua LinkedIn, email cá nhân, hoặc gọi điện — hệ thống **không** gửi tự động trong workflow này.
4. **Log Interaction:** BD rep ghi nhận kết quả tương tác (kênh, hướng, sentiment). Hệ thống cập nhật outreach state: COLD → NO_REPLY → REPLIED → POST_MEETING.
5. **Feedback:** BD rep đánh giá chất lượng AI draft (dùng nguyên, sửa, tự viết, bỏ qua) — giúp hệ thống cải thiện gợi ý.
6. **Background Recalculation:** Worker định kỳ tính lại trạng thái outreach cho contacts đang ở trạng thái WAIT, tự động cập nhật next step và follow-up count.

**So sánh 2 workflow:**
| | Automated Campaign (4a) | Manual Outreach (4b) |
|---|---|---|
| **Kênh** | Email (Gmail API) | LinkedIn, email thủ công, phone |
| **Ai gửi?** | Hệ thống tự động | BD rep gửi thủ công |
| **Nội dung** | Email templates sinh hàng loạt | Draft cá nhân hóa per-contact |
| **Follow-up** | Tự động theo cadence | AI gợi ý, BD rep quyết định |
| **Tracking** | Task status (queued→succeeded) | Interaction log + outreach state |

---

## 5. Database ER Diagram (Sơ đồ thực thể - quan hệ)

Diagram này thể hiện các entity chính trong database và mối quan hệ giữa chúng.

```mermaid
erDiagram
    USERS ||--o{ ORGANIZATIONS : "belongs to"
    USERS {
        uuid id PK
        string email
        string name
        string staff_email
        string google_id
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }

    ORGANIZATIONS ||--o{ AGENTS : "owns"
    ORGANIZATIONS ||--o{ CONTACTS : "owns"
    ORGANIZATIONS ||--o{ CAMPAIGNS : "owns"
    ORGANIZATIONS {
        uuid id PK
        string name
        string domain
        jsonb settings
        timestamp created_at
    }

    ROLES {
        uuid id PK
        uuid user_id FK
        uuid organization_id FK
        string role
    }
    USERS ||--o{ ROLES : "has"
    ORGANIZATIONS ||--o{ ROLES : "defines"

    CONTACTS {
        uuid id PK
        uuid organization_id FK
        string name
        string status
        string source
        jsonb information
        jsonb profile
        jsonb confirmed_facts
        jsonb ai_insights
        jsonb scores
        string outreach_stage
        string next_step
        string lifecycle_stage
        string business_stage
        timestamp deleted_at
        timestamp created_at
    }

    AGENTS ||--o{ CONVERSATIONS : "manages"
    AGENTS {
        uuid id PK
        uuid organization_id FK
        uuid user_id FK
        string name
        string email
        string email_provider
        jsonb credentials
        string status
        int daily_email_limit
        int emails_sent_today
        timestamp created_at
    }

    CAMPAIGNS ||--|{ TEMPLATES : "uses"
    CAMPAIGNS ||--o{ TASKS : "generates"
    CAMPAIGNS {
        uuid id PK
        uuid organization_id FK
        uuid agent_id FK
        string name
        string status
        jsonb outreach_strategy
        jsonb client_metadata
        jsonb playbook
        jsonb follow_up_schedule
        timestamp deleted_at
        timestamp created_at
    }

    LIST_CONTACTS ||--o{ CONTACTS : "groups"
    LIST_CONTACTS {
        uuid id PK
        uuid organization_id FK
        string name
        string source
        int contact_count
        timestamp created_at
    }

    CAMPAIGNS }o--|| LIST_CONTACTS : "targets"

    TEMPLATES {
        uuid id PK
        uuid campaign_id FK
        string name
        string subject
        text body
        string category
        int sequence_order
        timestamp deleted_at
        timestamp created_at
    }

    TASKS {
        uuid id PK
        uuid campaign_id FK
        uuid agent_id FK
        uuid contact_id FK
        string type
        string status
        jsonb payload
        jsonb result
        timestamp scheduled_at
        timestamp completed_at
        timestamp created_at
    }

    EMAILS {
        uuid id PK
        uuid agent_id FK
        uuid contact_id FK
        uuid conversation_id FK
        string subject
        text body
        string direction
        string status
        timestamp sent_at
        timestamp created_at
    }

    CONVERSATIONS {
        uuid id PK
        uuid agent_id FK
        uuid contact_id FK
        string type
        string status
        timestamp created_at
    }

    CUSTOM_FIELDS {
        uuid id PK
        uuid organization_id FK
        string name
        string field_type
        jsonb options
        timestamp created_at
    }

    INBOUND_LEAD_FORMS {
        uuid id PK
        uuid organization_id FK
        string name
        string identifier
        jsonb fields
        string status
        timestamp created_at
    }

    INTERACTIONS {
        uuid id PK
        uuid contact_id FK
        string type
        string channel
        jsonb metadata
        timestamp occurred_at
        timestamp created_at
    }
    CONTACTS ||--o{ INTERACTIONS : "has"

    SEGMENTATIONS {
        uuid id PK
        uuid organization_id FK
        string name
        jsonb criteria
        jsonb icp_definition
        timestamp created_at
    }
    ORGANIZATIONS ||--o{ SEGMENTATIONS : "defines"

    CONTACTS ||--o{ EMAILS : "receives"
    AGENTS ||--o{ EMAILS : "sends"
    CONTACTS ||--o{ TASKS : "assigned"
    AGENTS ||--o{ TASKS : "executes"
    ORGANIZATIONS ||--o{ CUSTOM_FIELDS : "defines"
    ORGANIZATIONS ||--o{ INBOUND_LEAD_FORMS : "owns"
```

**Mô tả:**
- **Organization** là đơn vị multi-tenant chính, sở hữu tất cả dữ liệu (contacts, campaigns, agents).
- **User** thuộc về organization thông qua **Role** (Admin/Member).
- **Contact** lưu profile + AI insights + scoring dưới dạng JSONB, theo dõi outreach_stage (COLD → NO_REPLY → REPLIED → POST_MEETING) và next_step.
- **Campaign** liên kết với ListContact (danh sách mục tiêu), Agent (thực thi), và Templates (nội dung email).
- **Task** là đơn vị thực thi nhỏ nhất: mỗi email gửi, mỗi follow-up đều là 1 task với explicit status (queued → processing → succeeded/failed).
- **Conversation** lưu lịch sử trao đổi giữa Agent và Contact; **Email** lưu từng email cụ thể.
- **Interaction** ghi nhận mọi tương tác (email open, reply, click, LinkedIn accept).
- **Segmentation** phân nhóm contact động theo criteria JSONB.
- **Custom Fields** cho phép mở rộng schema contact mà không cần migration.

---

## 6. Module Component Diagram (Sơ đồ module chức năng)

Diagram này thể hiện các module chức năng của hệ thống và mối liên hệ giữa chúng.

```mermaid
graph TB
    subgraph CoreModules ["Core Modules (Implemented & Active)"]
        Auth["🔐 Authentication<br/>Google OAuth · JWT<br/>Login · Refresh · Logout"]
        UserMgmt["👤 User Management<br/>Profile · API Keys"]
        OrgMgmt["🏢 Organization<br/>Create · Members · Roles"]
        ContactMgmt["📇 Contact Management<br/>CRUD · Search · Custom Fields<br/>Import (CSV · HubSpot · Apollo)"]
        ListMgmt["📋 Contact Lists<br/>Grouping · Association"]
        CampaignMgmt["📧 Campaign Management<br/>Create · Outreach Strategy<br/>Template Generation · Assign Agent"]
        AgentMgmt["🤖 Agent Management<br/>Gmail/Outlook OAuth<br/>Credentials · Daily Limits"]
        TaskMgmt["⚡ Task Orchestration<br/>Create · Enqueue · Status<br/>Send Email · Execute Campaign"]
        TemplateMgmt["📝 Template Management<br/>CRUD · Sequence Order"]
        CustomFields["🔧 Custom Fields<br/>Dynamic Schema Extension"]
        LeadForms["📥 Inbound Lead Forms<br/>Form Builder · Submit"]
    end

    subgraph BuiltNotWired ["Built — Not Yet Wired to Routes"]
        Playbook["📖 Playbook<br/>Multi-stage sequences<br/>Nurture · Outreach · Re-engagement"]
        Automation["⚙️ Automation Rules<br/>Trigger conditions<br/>Fit score thresholds"]
        Enrollment["🎯 Enrollment<br/>Contact → Playbook<br/>Approval workflows"]
        Segmentation["📊 Segmentation<br/>Dynamic grouping<br/>ICP scoring"]
        Intelligence["🧠 Intelligence<br/>Meeting briefs<br/>Scenario analysis"]
        Outreach["💬 Outreach Suggestions<br/>Contact suggestions<br/>Draft generation"]
    end

    subgraph BackgroundWorkers ["Background Workers (Asynq)"]
        EmailWorker["📤 Email Worker<br/>Send · Process Incoming"]
        CampaignWorker["🚀 Campaign Worker<br/>Execute sequences"]
        AgentWorker["🔄 Agent Worker<br/>Sync · Refresh tokens"]
        AIWorker["🤖 AI Worker<br/>Generate emails · Embeddings"]
    end

    Auth --> UserMgmt
    UserMgmt --> OrgMgmt
    OrgMgmt --> ContactMgmt
    ContactMgmt --> ListMgmt
    ListMgmt --> CampaignMgmt
    CampaignMgmt --> AgentMgmt
    CampaignMgmt --> TemplateMgmt
    CampaignMgmt --> TaskMgmt
    ContactMgmt --> CustomFields
    ContactMgmt --> LeadForms

    TaskMgmt --> EmailWorker
    TaskMgmt --> CampaignWorker
    TaskMgmt --> AgentWorker
    CampaignMgmt --> AIWorker

    Playbook --> Automation
    Automation --> Enrollment
    Segmentation --> Automation
    Intelligence --> Outreach
    Outreach --> CampaignMgmt

    style Auth fill:#4CAF50,color:#fff
    style UserMgmt fill:#4CAF50,color:#fff
    style OrgMgmt fill:#4CAF50,color:#fff
    style ContactMgmt fill:#4CAF50,color:#fff
    style ListMgmt fill:#4CAF50,color:#fff
    style CampaignMgmt fill:#4CAF50,color:#fff
    style AgentMgmt fill:#4CAF50,color:#fff
    style TaskMgmt fill:#4CAF50,color:#fff
    style TemplateMgmt fill:#4CAF50,color:#fff
    style CustomFields fill:#4CAF50,color:#fff
    style LeadForms fill:#4CAF50,color:#fff

    style Playbook fill:#FF9800,color:#fff
    style Automation fill:#FF9800,color:#fff
    style Enrollment fill:#FF9800,color:#fff
    style Segmentation fill:#FF9800,color:#fff
    style Intelligence fill:#FF9800,color:#fff
    style Outreach fill:#FF9800,color:#fff
```

**Mô tả:**
- **Xanh lá (Core Modules):** 11 module đã implement đầy đủ và đang hoạt động — từ Authentication đến Task Orchestration.
- **Cam (Built — Not Wired):** 6 module đã code backend handler nhưng chưa đăng ký routes — bao gồm Playbook (kịch bản tự động), Automation Rules (trigger điều kiện), Enrollment (gán contact vào playbook), Segmentation (phân nhóm), Intelligence (AI insights), và Outreach Suggestions (gợi ý contact + draft).
- **Background Workers:** 4 loại worker xử lý task nền — Email (gửi/nhận), Campaign (thực thi sequence), Agent (sync credentials), AI (sinh nội dung).
- Luồng chính: Auth → User → Org → Contact → List → Campaign → Agent → Task → Workers.

---

## 7. Authentication Flow (Luồng xác thực)

```mermaid
sequenceDiagram
    actor User
    participant Browser
    participant FE as Frontend<br/>(Next.js)
    participant MW as Auth Middleware
    participant API as Backend API
    participant Google as Google OAuth
    participant DB as PostgreSQL

    Note over User,DB: Login Flow
    User->>Browser: Navigate to /auth/login
    Browser->>FE: Load login page
    User->>FE: Click "Login with Google"
    FE->>Google: Redirect to Google OAuth
    Google-->>User: Google consent screen
    User->>Google: Approve
    Google-->>FE: Redirect with auth code
    FE->>API: POST /v1/auth/login (auth code)
    API->>Google: Exchange code for tokens
    Google-->>API: Access token + ID token
    API->>DB: Find/Create user
    API->>API: Generate JWT (access + refresh)
    API-->>FE: JWT access_token + refresh_token
    FE->>FE: Store tokens in cookies (encrypted)

    Note over User,DB: Protected Route Access
    User->>Browser: Navigate to /campaigns
    Browser->>MW: Request with JWT cookie
    MW->>MW: Validate JWT + check expiry
    MW->>API: GET /v1/users/me (verify user)
    API-->>MW: User data + organization
    MW-->>Browser: Allow access

    Note over User,DB: Token Refresh
    FE->>API: Request fails with 401
    FE->>API: POST /v1/auth/refresh (refresh_token)
    API->>API: Validate refresh token
    API->>API: Generate new access_token
    API-->>FE: New access_token
    FE->>FE: Update cookie
    FE->>API: Retry original request
```

**Mô tả:**
- **Login:** Người dùng chọn "Login with Google" → redirect đến Google OAuth → nhận auth code → backend exchange code lấy token → tạo/tìm user trong DB → trả JWT access/refresh token → frontend lưu vào cookie (mã hóa).
- **Protected Access:** Mỗi request tới protected route, Next.js middleware validate JWT, kiểm tra expiry, và verify user qua backend API.
- **Token Refresh:** Khi access token hết hạn (401), frontend tự động gọi refresh endpoint, nhận token mới, và retry request gốc — người dùng không bị gián đoạn.

---

## 8. Task Orchestration Flow (Luồng điều phối Task)

```mermaid
stateDiagram-v2
    [*] --> Queued: Task Enqueued<br/>(via API or Worker)

    Queued --> Processing: Worker picks up task

    Processing --> Succeeded: Execution completed
    Processing --> Failed: Error occurred
    Processing --> Retrying: Transient error

    Retrying --> Processing: Retry attempt
    Retrying --> Failed: Max retries exceeded

    Failed --> [*]: Task archived
    Succeeded --> [*]: Task archived

    Succeeded --> NewTask: Triggers next task<br/>(e.g., follow-up)

    NewTask --> Queued

    state Processing {
        [*] --> LoadData: Read from DB
        LoadData --> Execute: Call external API<br/>(Gmail, OpenAI)
        Execute --> SaveResult: Write to DB
        SaveResult --> [*]
    }
```

**Mô tả:**
COSMO sử dụng mô hình task queue để đảm bảo mọi campaign step được thực thi tin cậy:
- **Queued:** Task được tạo qua API (`/v1/tasks/enqueue/*`) hoặc bởi worker khác và đẩy vào Redis queue.
- **Processing:** Asynq worker poll task, load dữ liệu từ DB, gọi external API (gửi email qua Gmail, sinh nội dung qua OpenAI), và lưu kết quả.
- **Succeeded/Failed:** Kết quả được ghi rõ ràng. Task thành công có thể trigger task tiếp theo (ví dụ: gửi email xong → schedule follow-up).
- **Retrying:** Lỗi tạm thời (network timeout, rate limit) được retry tự động. Vượt quá max retries → Failed.
- Mô hình này đảm bảo **traceability** (mọi action đều có log) và **reliability** (retry tự động, không mất task).

---

## 9. Use Case Diagram — Overall (Sơ đồ use case tổng quan)

Mở file `usecase-overall.drawio` bằng https://app.diagrams.net/ hoặc VS Code extension "Draw.io Integration".

**Mô tả:**
Sơ đồ use case tổng quan thể hiện toàn bộ chức năng chính của hệ thống COSMO ở mức module, phân theo 2 phase triển khai.

- **5 Actor:**
  - **Admin** — BD rep có role `admin` trong organization. Có toàn quyền trên hệ thống bao gồm quản lý organization settings, thêm/xóa thành viên, và gán roles. Admin được tự động gán khi tạo organization mới.
  - **BD Rep (Member)** — BD rep có role `member`. Thực hiện được mọi chức năng nghiệp vụ (quản lý contacts, campaigns, agents, outreach) trừ quản lý organization settings và members.
  - **External Visitor** — Người ngoài hệ thống (chưa đăng nhập). Chỉ có thể submit inbound lead form công khai qua public URL.
  - **Background Worker (Asynq)** — System actor hoạt động tự động. Xử lý task queue (gửi email, execute campaign), recalculate outreach state, tính lại segment scores, và execute playbook enrollment stages.
  - **AI Tool Agent (MCP)** — External AI tool gọi API qua Model Context Protocol để tạo campaign kèm contacts trong 1 lần gọi duy nhất.

- **Phase 1 — Core Platform (xanh lá):** 8 use case module đã triển khai đầy đủ với API routes đã đăng ký và frontend UI hoạt động: Authentication & Account, Organization, Contacts & Custom Fields, Contact Lists, Campaigns & Templates, Agents, Task Orchestration, Inbound Lead Forms.

- **Phase 2 — AI-Assisted Outreach (cam):** 6 use case module đã code backend hoàn chỉnh (handlers, services, domain models, database migrations, background workers) chờ kích hoạt API routes: AI Contact Intelligence, AI-Assisted Manual Outreach, Meetings, Segments, Playbook Automation, MCP Integration.

- **Phân quyền:** Admin và Member chia sẻ hầu hết use case. Sự khác biệt duy nhất: chỉ Admin mới được Manage Organization (update settings, manage members & roles). Tất cả các chức năng nghiệp vụ BD khác đều mở cho cả 2 role.
