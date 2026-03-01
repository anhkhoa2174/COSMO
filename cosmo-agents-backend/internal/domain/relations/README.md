# Relations Package

This package provides a solution to circular import dependencies in the domain layer by defining composite structs that combine base domain models with their relationships.

## Problem

When domain models reference each other directly, it creates circular import dependencies:

```go
// agent.go
type Agent struct {
    // ...
    Conversations []conversation.Conversation  // ❌ Circular import
    User          *user.User                  // ❌ Circular import
}

// conversation.go
type Conversation struct {
    // ...
    Agent *agent.Agent  // ❌ Circular import
}
```

## Solution

The relations package provides composite structs that combine base models with their relationships:

```go
// relations.go
type AgentWithConversations struct {
    agent.Agent
    Conversations []conversation.Conversation `gorm:"foreignKey:AgentID"`
}

type ConversationWithAgent struct {
    conversation.Conversation
    Agent *agent.Agent `gorm:"foreignKey:AgentID"`
}
```

## Usage

### Repository Pattern

```go
type AgentRepository struct {
    db *gorm.DB
}

func (r *AgentRepository) GetAgentWithConversations(id uuid.UUID) (*relations.AgentWithConversations, error) {
    var agent relations.AgentWithConversations
    err := r.db.Preload("Conversations").First(&agent, "id = ?", id).Error
    return &agent, err
}
```

### Service Layer

```go
func (s *AgentService) GetAgentDashboard(id uuid.UUID) (*relations.AgentWithFullRelations, error) {
    agent, err := s.repo.GetAgentWithFullRelations(id)
    if err != nil {
        return nil, err
    }

    // Access all related data:
    // agent.User
    // agent.Organization
    // agent.Conversations

    return agent, nil
}
```

## Available Relations

### Agent Relations
- `AgentWithConversations` - Agent with their conversations
- `AgentWithUser` - Agent with user information
- `AgentWithOrganization` - Agent with organization information
- `AgentWithFullRelations` - Agent with all relationships

### Conversation Relations
- `ConversationWithAgent` - Conversation with agent information
- `ConversationWithEmails` - Conversation with all emails
- `ConversationWithCampaign` - Conversation with campaign information
- `ConversationWithFullRelations` - Conversation with all relationships

### Email Relations
- `EmailWithConversation` - Email with conversation information
- `EmailWithCampaign` - Email with campaign information
- `EmailWithFullRelations` - Email with all relationships

### User Relations
- `UserWithOrganizations` - User with their organizations
- `UserWithRoles` - User with their roles
- `UserWithNotifications` - User with their notifications
- `UserWithAgents` - User with their agents
- `UserWithFullRelations` - User with all relationships

### Organization Relations
- `OrganizationWithUser` - Organization with user information
- `OrganizationWithRoles` - Organization with roles
- `OrganizationWithAgents` - Organization with agents
- `OrganizationWithFullRelations` - Organization with all relationships

### Campaign Relations
- `CampaignWithAgent` - Campaign with agent information
- `CampaignWithEmails` - Campaign with emails
- `CampaignWithConversations` - Campaign with conversations
- `CampaignWithFullRelations` - Campaign with all relationships

## Migration Steps

1. **Remove direct relationships** from base domain models (already done):
   ```go
   // Before (causes circular import)
   type Agent struct {
       User *user.User  // ❌ Remove this
   }

   // After (clean)
   type Agent struct {
       // Only fields, no direct relationships
   }
   ```

2. **Use relations package** for queries:
   ```go
   // Before
   var agent agent.Agent
   db.Preload("User").Preload("Conversations").First(&agent)

   // After
   var agent relations.AgentWithFullRelations
   db.Preload("User").Preload("Conversations").First(&agent)
   ```

3. **Update API responses**:
   ```go
   // API handlers can now return full relations
   func (h *AgentHandler) GetAgent(c *gin.Context) {
       agent, err := h.repo.GetAgentWithFullRelations(id)
       c.JSON(200, agent)  // Returns agent with all relationships
   }
   ```

## Benefits

1. **No circular imports** - Domain models remain independent
2. **Type safety** - All relationships are properly typed
3. **Flexible querying** - Choose exactly which relationships to load
4. **Clean separation** - Base models focus on data, relations focus on connections
5. **GORM compatible** - Works seamlessly with GORM's preload system

## Best Practices

1. **Use specific relations** when possible (e.g., `AgentWithUser`) instead of full relations for better performance
2. **Lazy load** expensive relationships when not immediately needed
3. **Create custom relations** for specific use cases not covered by the standard ones
4. **Document** which relations are used in each API endpoint for maintainability