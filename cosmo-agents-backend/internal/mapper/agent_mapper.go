package mapper

import (
	"github.com/google/uuid"
	agentUseCase "github.com/rockship/cosmo-agents-go/internal/usecase/agent"
)

// AgentCreateRequestToUseCase converts HTTP request to use case.
func AgentCreateRequestToUseCase(req *AgentCreateRequest) *agentUseCase.CreateAgentUseCase {
	return &agentUseCase.CreateAgentUseCase{
		UserID:         req.UserID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Email:          req.Email,
		EmailProvider:  req.EmailProvider,
		Persona:        req.Persona,
		Signature:      req.Signature,
		Picture:        req.Picture,
		Credentials:    req.Credentials,
		DailyLimit:     req.DailyLimit,
		MaxDailyLimit:  req.MaxDailyLimit,
		Metadata:       req.Metadata,
	}
}

// AgentUpdateRequestToUseCase converts HTTP request to use case.
func AgentUpdateRequestToUseCase(agentID string, req *AgentUpdateRequest) (*agentUseCase.UpdateAgentUseCase, error) {
	id, err := ParseUUID(agentID)
	if err != nil {
		return nil, err
	}

	return &agentUseCase.UpdateAgentUseCase{
		AgentID:       id,
		UserID:        req.UserID,
		Name:          req.Name,
		Email:         req.Email,
		Status:        req.Status,
		EmailProvider: req.EmailProvider,
		Persona:       req.Persona,
		Signature:     req.Signature,
		Picture:       req.Picture,
		Credentials:   req.Credentials,
		DailyLimit:    req.DailyLimit,
		MaxDailyLimit: req.MaxDailyLimit,
		ValidCred:     req.ValidCred,
		LastHistoryID: req.LastHistoryID,
		Metadata:      req.Metadata,
	}, nil
}

// AgentGetRequestToUseCase converts HTTP request to use case.
func AgentGetRequestToUseCase(agentID string, userID *string) (*agentUseCase.GetAgentUseCase, error) {
	id, err := ParseUUID(agentID)
	if err != nil {
		return nil, err
	}

	var uid *uuid.UUID
	if userID != nil {
		parsedID, err := ParseUUID(*userID)
		if err != nil {
			return nil, err
		}
		uid = &parsedID
	}

	return &agentUseCase.GetAgentUseCase{
		AgentID: id,
		UserID:  uid,
	}, nil
}

// AgentDeleteRequestToUseCase converts HTTP request to use case.
func AgentDeleteRequestToUseCase(agentID string, userID *string) (*agentUseCase.DeleteAgentUseCase, error) {
	id, err := ParseUUID(agentID)
	if err != nil {
		return nil, err
	}

	var uid *uuid.UUID
	if userID != nil {
		parsedID, err := ParseUUID(*userID)
		if err != nil {
			return nil, err
		}
		uid = &parsedID
	}

	return &agentUseCase.DeleteAgentUseCase{
		AgentID: id,
		UserID:  uid,
	}, nil
}

// AgentGetByUserRequestToUseCase converts HTTP request to use case.
func AgentGetByUserRequestToUseCase(userID string) (*agentUseCase.GetAgentsByUserUseCase, error) {
	id, err := ParseUUID(userID)
	if err != nil {
		return nil, err
	}

	return &agentUseCase.GetAgentsByUserUseCase{
		UserID: id,
	}, nil
}

// AgentResponseToHTTP converts use case response to HTTP response.
func AgentResponseToHTTP(resp *agentUseCase.AgentResponse) *AgentResponse {
	return &AgentResponse{
		ID:              resp.ID,
		UserID:          resp.UserID,
		OrganizationID:  resp.OrganizationID,
		Name:            resp.Name,
		Email:           resp.Email,
		Status:          resp.Status,
		EmailProvider:   resp.EmailProvider,
		Signature:       resp.Signature,
		Picture:         resp.Picture,
		Persona:         resp.Persona,
		LastHistoryID:   resp.LastHistoryID,
		DailyLimit:      resp.DailyLimit,
		MaxDailyLimit:   resp.MaxDailyLimit,
		ValidCred:       resp.ValidCred,
		EmailsSentToday: resp.EmailsSentToday,
		Metadata:        resp.Metadata,
		CreatedAt:       resp.CreatedAt,
		UpdatedAt:       resp.UpdatedAt,
	}
}

// AgentsListResponseToHTTP converts use case list response to HTTP response.
func AgentsListResponseToHTTP(resp *agentUseCase.AgentsListResponse) *AgentsListResponse {
	agents := make([]*AgentResponse, len(resp.Agents))
	for i, agent := range resp.Agents {
		agents[i] = AgentResponseToHTTP(&agent)
	}

	return &AgentsListResponse{
		Agents: agents,
		Total:  resp.Total,
	}
}
