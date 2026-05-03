package mcp

// Request is a JSON-RPC request used by MCP over WebSocket.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response is a JSON-RPC response used by MCP over WebSocket.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error describes a JSON-RPC error payload.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeRequest is the MCP initialize request payload.
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
	ClientInfo      ClientInfo             `json:"clientInfo,omitempty"`
}

// ClientInfo describes the connecting MCP client.
type ClientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// InitializeResult is the MCP initialize response payload.
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

// ServerInfo describes the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool describes an MCP tool.
type Tool struct {
	Name               string                 `json:"name"`
	Title              string                 `json:"title,omitempty"`
	Description        string                 `json:"description"`
	Category           string                 `json:"category,omitempty"`
	SourceType         string                 `json:"sourceType,omitempty"`
	Enabled            bool                   `json:"enabled,omitempty"`
	RequiresMembership bool                   `json:"requiresMembership,omitempty"`
	CanUse             bool                   `json:"canUse,omitempty"`
	InputSchema        map[string]interface{} `json:"inputSchema"`
	OutputSchema       map[string]interface{} `json:"outputSchema,omitempty"`
}

// ListToolsResult contains the available tools.
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest is the request payload for tools/call.
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult is the result payload for tools/call.
type CallToolResult struct {
	Content           []Content              `json:"content,omitempty"`
	StructuredContent map[string]interface{} `json:"structuredContent,omitempty"`
	IsError           bool                   `json:"isError,omitempty"`
}

// Content represents MCP tool output content.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
