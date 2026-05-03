package mcp

import (
	"context"

	"brights/api/internal/domain"
)

func EndpointToolsForLearner(ctx context.Context, server *Server, learnerID uint, endpointID uint) (domain.MCPEndpoint, []Tool, error) {
	endpoint, err := server.service.GetLearnerMCPEndpoint(ctx, learnerID, endpointID)
	if err != nil {
		return domain.MCPEndpoint{}, nil, err
	}

	return endpoint, server.tools(), nil
}
