package mcp

import (
	"context"
	"strings"

	"brights/api/internal/domain"
)

func EndpointToolsForLearner(ctx context.Context, server *Server, learnerID uint, endpointID uint, subjectKey string) (domain.MCPEndpoint, []Tool, error) {
	endpoint, err := server.service.GetLearnerMCPEndpoint(ctx, learnerID, endpointID)
	if err != nil {
		return domain.MCPEndpoint{}, nil, err
	}

	learner, err := server.service.GetLearnerByID(ctx, learnerID)
	if err != nil {
		return domain.MCPEndpoint{}, nil, err
	}

	session := &Session{
		UserID:     learner.ID,
		Username:   learner.Username,
		SubjectKey: strings.TrimSpace(subjectKey),
	}
	return endpoint, server.tools(ctx, session), nil
}
