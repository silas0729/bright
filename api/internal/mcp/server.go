package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"brights/api/internal/domain"
	"brights/api/internal/service"
	"brights/api/internal/userauth"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "brights-mcp"
	serverVersion   = "0.1.0"
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	maxMessageSize  = 1024 * 1024
)

var (
	toolResultSchema = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"success": map[string]interface{}{"type": "boolean"},
			"tool":    map[string]interface{}{"type": "string"},
			"result":  map[string]interface{}{},
			"error":   map[string]interface{}{"type": "string"},
		},
		"required": []string{"success", "tool"},
	}
)

// Server exposes Brights data through a lightweight MCP websocket endpoint.
type Server struct {
	service  *service.Service
	userAuth *userauth.Manager
	upgrader websocket.Upgrader
	writeMu  sync.Mutex
}

// NewServer creates an MCP server instance.
func NewServer(svc *service.Service, userAuth *userauth.Manager) *Server {
	return &Server{
		service:  svc,
		userAuth: userAuth,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// HandleWebSocket upgrades the HTTP request and serves MCP messages.
func (s *Server) HandleWebSocket(c *gin.Context) {
	session, err := s.authenticateRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Gin writes headers lazily, so we upgrade directly from the wrapped writer/request.
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	done := make(chan struct{})
	go s.keepAlive(conn, done)
	defer close(done)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(payload, &req); err != nil {
			s.writeResponse(conn, Response{
				JSONRPC: "2.0",
				Error: &Error{
					Code:    -32700,
					Message: "parse error",
					Data:    err.Error(),
				},
			})
			continue
		}

		resp := s.handleRequest(c.Request.Context(), session, req)
		if req.ID == nil && req.Method == "notifications/initialized" {
			continue
		}
		if err := s.writeResponse(conn, resp); err != nil {
			return
		}
	}
}

// HandleInfo returns a simple HTTP description of the MCP endpoint.
func (s *Server) HandleInfo(c *gin.Context) {
	host := strings.TrimSpace(c.Request.Host)
	scheme := "ws"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "wss"
	}

	exampleSubject := firstNonEmpty(strings.TrimSpace(c.Query("subject")), "english")
	websocketURL := fmt.Sprintf("%s://%s/mcp?subject=%s&token={learner_access_token}", scheme, host, exampleSubject)
	session, err := s.authenticateOptionalRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	toolItems := s.tools(c.Request.Context(), session)

	c.JSON(http.StatusOK, gin.H{
		"name":             serverName,
		"version":          serverVersion,
		"protocolVersion":  protocolVersion,
		"websocketPath":    "/mcp",
		"websocketURL":     websocketURL,
		"availableMethods": []string{"initialize", "ping", "tools/list", "tools/call"},
		"toolCount":        len(toolItems),
		"tools":            toolItems,
		"auth": gin.H{
			"mode":                 "learner_bearer_or_query_token",
			"queryTokenParam":      "token",
			"querySubjectParam":    "subject",
			"tokenOptionalForInfo": true,
			"requiresMembership":   true,
		},
		"viewer": gin.H{
			"isAuthenticated": session != nil && strings.TrimSpace(session.Username) != "",
			"username":        firstNonEmpty(session.Username, ""),
			"subjectKey":      firstNonEmpty(session.SubjectKey, exampleSubject),
		},
		"examples": gin.H{
			"headerAuthURL": websocketURL[:strings.LastIndex(websocketURL, "&token=")],
			"queryAuthURL":  websocketURL,
			"initialize": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params": map[string]interface{}{
					"protocolVersion": protocolVersion,
					"capabilities":    map[string]interface{}{},
					"clientInfo": map[string]interface{}{
						"name":    "brights-demo-client",
						"version": "1.0.0",
					},
				},
			},
			"listTools": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "tools/list",
			},
			"callTool": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      3,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "search_words",
					"arguments": map[string]interface{}{
						"subject_key": exampleSubject,
						"query":       "travel",
						"page":        1,
						"page_size":   10,
					},
				},
			},
		},
	})
}

// HandleToolMarket returns a paged MCP tool market response for the public site.
func (s *Server) HandleToolMarket(c *gin.Context) {
	session, err := s.authenticateOptionalRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	c.JSON(http.StatusOK, s.toolMarket(c.Request.Context(), session, strings.TrimSpace(c.Query("q")), strings.TrimSpace(c.Query("category")), page, pageSize))
}

func (s *Server) keepAlive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := s.writeMessage(conn, websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, session Session, req Request) Response {
	switch req.Method {
	case "initialize":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: protocolVersion,
				Capabilities: map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
					"experimental": map[string]interface{}{
						"structuredContent": true,
					},
				},
				ServerInfo: ServerInfo{
					Name:    serverName,
					Version: serverVersion,
				},
			},
		}
	case "ping":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	case "tools/list":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ListToolsResult{
				Tools: s.tools(ctx, &session),
			},
		}
	case "tools/call":
		params, err := decodeCallToolRequest(req.Params)
		if err != nil {
			return s.invalidParams(req.ID, err)
		}
		effectiveSession := sessionForToolCall(session, params)
		result, err := s.callTool(ctx, effectiveSession, params)
		if err != nil {
			if isUnknownToolError(err) {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &Error{
						Code:    -32601,
						Message: "tool not found",
						Data:    err.Error(),
					},
				}
			}
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  newToolErrorResult(params.Name, err),
			}
		}
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
	case "notifications/initialized":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}}
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32601,
				Message: "method not found",
				Data:    req.Method,
			},
		}
	}
}

type Session struct {
	UserID      uint
	Username    string
	SubjectKey  string
	Token       string
	HTTPBaseURL string
}

func (s *Server) authenticateRequest(c *gin.Context) (Session, error) {
	if s.userAuth == nil {
		return Session{}, errors.New("user auth manager is not configured")
	}

	subjectKey := firstNonEmpty(c.Query("subject"), c.Query("subject_key"))
	token := tokenFromRequest(c.Request)
	if token == "" {
		return Session{}, errors.New("missing learner access token")
	}

	claims, err := s.userAuth.ParseToken(token)
	if err != nil {
		return Session{}, err
	}

	return Session{
		UserID:      claims.UserID,
		Username:    claims.Username,
		SubjectKey:  strings.TrimSpace(subjectKey),
		Token:       token,
		HTTPBaseURL: requestHTTPBaseURL(c.Request),
	}, nil
}

func (s *Server) authenticateOptionalRequest(c *gin.Context) (*Session, error) {
	subjectKey := firstNonEmpty(c.Query("subject"), c.Query("subject_key"))
	token := tokenFromRequest(c.Request)
	if token == "" {
		return &Session{
			SubjectKey:  strings.TrimSpace(subjectKey),
			HTTPBaseURL: requestHTTPBaseURL(c.Request),
		}, nil
	}
	session, err := s.authenticateRequest(c)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func sessionForToolCall(session Session, req CallToolRequest) Session {
	if req.Arguments == nil {
		return session
	}

	subjectKey := firstNonEmpty(
		stringArg(req.Arguments, "subject_key", ""),
		stringArg(req.Arguments, "subject", ""),
		session.SubjectKey,
	)
	session.SubjectKey = subjectKey
	return session
}

func tokenFromRequest(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	queryToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if queryToken != "" {
		return queryToken
	}
	return ""
}

func (s *Server) invalidParams(id interface{}, err error) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    -32602,
			Message: "invalid params",
			Data:    err.Error(),
		},
	}
}

func (s *Server) writeResponse(conn *websocket.Conn, resp Response) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteJSON(resp)
}

func (s *Server) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(messageType, data)
}

func decodeCallToolRequest(params interface{}) (CallToolRequest, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return CallToolRequest{}, err
	}
	var request CallToolRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return CallToolRequest{}, err
	}
	var rawParams map[string]interface{}
	if err := json.Unmarshal(raw, &rawParams); err == nil {
		if v, ok := rawParams["arguments"]; ok {
			parsedArgs, parseErr := normalizeArguments(v)
			if parseErr != nil {
				return CallToolRequest{}, parseErr
			}
			request.Arguments = parsedArgs
		}
	}
	if strings.TrimSpace(request.Name) == "" {
		return CallToolRequest{}, fmt.Errorf("tool name is required")
	}
	if request.Arguments == nil {
		request.Arguments = map[string]interface{}{}
	}
	return request, nil
}

func (s *Server) tools(ctx context.Context, session *Session) []Tool {
	tools := []Tool{
		{
			Name:         "list_subjects",
			Title:        "List Subjects",
			Description:  "List all Brights subjects.",
			Category:     "catalog",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_categories",
			Title:       "List Categories",
			Description: "List categories for a subject and kind.",
			Category:    "catalog",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string", "description": "Optional subject key, for example english."},
				"kind":        map[string]interface{}{"type": "string", "description": "Optional category kind, defaults to topic."},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "list_grades",
			Title:        "List Grades",
			Description:  "List all grade definitions.",
			Category:     "catalog",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "search_words",
			Title:       "Search Words",
			Description: "Search and paginate Brights words.",
			Category:    "catalog",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key":    map[string]interface{}{"type": "string"},
				"subject_id":     map[string]interface{}{"type": "integer"},
				"category_id":    map[string]interface{}{"type": "integer"},
				"classification": map[string]interface{}{"type": "string"},
				"grade_id":       map[string]interface{}{"type": "integer"},
				"query":          map[string]interface{}{"type": "string", "description": "Search keyword."},
				"q":              map[string]interface{}{"type": "string", "description": "Search keyword alias."},
				"page":           map[string]interface{}{"type": "integer"},
				"page_size":      map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_classification_stats",
			Title:       "List Classification Stats",
			Description: "List classification statistics with pagination.",
			Category:    "catalog",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "list_membership_plans",
			Title:        "List Membership Plans",
			Description:  "List Brights membership or payment plans.",
			Category:     "membership",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "get_catalog_stats",
			Title:        "Get Catalog Stats",
			Description:  "Get overall Brights catalog statistics.",
			Category:     "catalog",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "search_knowledge_base",
			Title:       "Search Knowledge Base",
			Description: "Search uploaded text or spreadsheet knowledge base content.",
			Category:    "knowledge",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"query":       map[string]interface{}{"type": "string", "description": "Knowledge base search keyword."},
				"q":           map[string]interface{}{"type": "string", "description": "Knowledge base search keyword alias."},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_my_knowledge_base_documents",
			Title:       "List My Knowledge Base Documents",
			Description: "List the current learner's uploaded knowledge base documents.",
			Category:    "knowledge",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"query":       map[string]interface{}{"type": "string"},
				"q":           map[string]interface{}{"type": "string"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "view_knowledge_base_document",
			Title:       "View Knowledge Base Document",
			Description: "View chunks and original content for one uploaded knowledge base document.",
			Category:    "knowledge",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"document_id": map[string]interface{}{"type": "integer"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_my_payment_orders",
			Title:       "List My Payment Orders",
			Description: "List the current learner's recharge or purchase records.",
			Category:    "account",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"status":      map[string]interface{}{"type": "string"},
				"query":       map[string]interface{}{"type": "string"},
				"q":           map[string]interface{}{"type": "string"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_my_memberships",
			Title:       "List My Memberships",
			Description: "List the current learner's membership records.",
			Category:    "account",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"status":      map[string]interface{}{"type": "string"},
				"query":       map[string]interface{}{"type": "string"},
				"q":           map[string]interface{}{"type": "string"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:         "get_invite_summary",
			Title:        "Get Invite Summary",
			Description:  "Get the current learner's invite code and invite statistics.",
			Category:     "account",
			SourceType:   "builtin",
			Enabled:      true,
			InputSchema:  objectSchema(nil),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "get_learning_summary",
			Title:       "Get Learning Summary",
			Description: "Get level counts, difficulty counts, and memory curve statistics for the current learner.",
			Category:    "learning",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "list_learning_progress",
			Title:       "List Learning Progress",
			Description: "List tracked learning words with level, difficulty, and next review time.",
			Category:    "learning",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"subject_key": map[string]interface{}{"type": "string"},
				"query":       map[string]interface{}{"type": "string"},
				"q":           map[string]interface{}{"type": "string"},
				"level":       map[string]interface{}{"type": "string"},
				"difficulty":  map[string]interface{}{"type": "string"},
				"due_only":    map[string]interface{}{"type": "boolean"},
				"page":        map[string]interface{}{"type": "integer"},
				"page_size":   map[string]interface{}{"type": "integer"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "save_learning_word_progress",
			Title:       "Save Learning Word Progress",
			Description: "Create or update the current learner's level and difficulty for a word.",
			Category:    "learning",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"word_id":     map[string]interface{}{"type": "integer"},
				"subject_key": map[string]interface{}{"type": "string"},
				"level":       map[string]interface{}{"type": "string"},
				"difficulty":  map[string]interface{}{"type": "string"},
			}),
			OutputSchema: toolResultSchema,
		},
		{
			Name:        "review_learning_word",
			Title:       "Review Learning Word",
			Description: "Record whether the current learner remembered a word and schedule the next review.",
			Category:    "learning",
			SourceType:  "builtin",
			Enabled:     true,
			InputSchema: objectSchema(map[string]interface{}{
				"word_id":     map[string]interface{}{"type": "integer"},
				"subject_key": map[string]interface{}{"type": "string"},
				"remembered":  map[string]interface{}{"type": "boolean"},
				"level":       map[string]interface{}{"type": "string"},
				"difficulty":  map[string]interface{}{"type": "string"},
			}),
			OutputSchema: toolResultSchema,
		},
	}
	tools = append(tools, xiaomiBuiltinTools()...)
	tools = append(tools, s.dynamicAPITools(ctx, session)...)

	configMap, err := s.service.GetMCPToolConfigMap(ctx)
	if err != nil {
		return tools
	}

	hasMembership := false
	if session != nil && strings.TrimSpace(session.Username) != "" && strings.TrimSpace(session.SubjectKey) != "" {
		if allowed, accessErr := s.subjectMembershipAccess(ctx, *session); accessErr == nil {
			hasMembership = allowed
		}
	}
	hasSession := session != nil && strings.TrimSpace(session.Username) != ""

	items := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		tool.RequiresAuth = toolRequiresAuthenticatedSession(tool)
		if config, ok := configMap[tool.Name]; ok {
			if strings.TrimSpace(config.Title) != "" {
				tool.Title = config.Title
			}
			if strings.TrimSpace(config.Description) != "" {
				tool.Description = config.Description
			}
			if strings.TrimSpace(config.Category) != "" {
				tool.Category = config.Category
			}
			if strings.TrimSpace(config.SourceType) != "" {
				tool.SourceType = config.SourceType
			}
			tool.Enabled = config.IsEnabled
			tool.RequiresMembership = config.RequiresMembership
		}
		if !tool.Enabled {
			continue
		}
		switch {
		case tool.RequiresAuth && !hasSession:
			tool.CanUse = false
		case tool.RequiresMembership:
			tool.CanUse = hasMembership
		default:
			tool.CanUse = true
		}
		items = append(items, tool)
	}

	return items
}

func objectSchema(properties map[string]interface{}) map[string]interface{} {
	if properties == nil {
		properties = map[string]interface{}{}
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
}

func (s *Server) callTool(ctx context.Context, session Session, req CallToolRequest) (CallToolResult, error) {
	canonicalName := canonicalToolName(req.Name)
	hasMembership, err := s.subjectMembershipAccess(ctx, session)
	if err != nil {
		return CallToolResult{}, err
	}
	configMap, err := s.service.GetMCPToolConfigMap(ctx)
	if err != nil {
		return CallToolResult{}, err
	}
	if config, ok := configMap[canonicalName]; ok {
		if !config.IsEnabled {
			return CallToolResult{}, fmt.Errorf("tool %s is disabled by administrator", canonicalName)
		}
		if config.RequiresMembership && !hasMembership {
			return CallToolResult{}, fmt.Errorf("membership required for tool %s", canonicalName)
		}
	}
	switch canonicalName {
	case "list_subjects":
		data, err := s.service.ListSubjects(ctx)
		return newToolResult(canonicalName, data, err)
	case "list_categories":
		subjectKey := stringArg(req.Arguments, "subject_key", session.SubjectKey)
		kind := stringArg(req.Arguments, "kind", "topic")
		data, err := s.service.ListCategories(ctx, subjectKey, kind)
		return newToolResult(canonicalName, data, err)
	case "list_grades":
		data, err := s.service.ListGrades(ctx)
		return newToolResult(canonicalName, data, err)
	case "search_words":
		data, err := s.service.ListWords(ctx, domain.WordFilter{
			SubjectKey:     stringArg(req.Arguments, "subject_key", session.SubjectKey),
			SubjectID:      uint(intArg(req.Arguments, "subject_id", 0)),
			CategoryID:     uint(intArg(req.Arguments, "category_id", 0)),
			Classification: stringArg(req.Arguments, "classification", ""),
			GradeID:        uint(intArg(req.Arguments, "grade_id", 0)),
			Query:          firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Page:           intArg(req.Arguments, "page", 1),
			PageSize:       intArg(req.Arguments, "page_size", 20),
			HideVIP:        !hasMembership,
		})
		return newToolResult(canonicalName, data, err)
	case "list_classification_stats":
		data, err := s.service.ListClassificationStatsPaged(ctx, domain.ClassificationStatFilter{
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Page:       intArg(req.Arguments, "page", 1),
			PageSize:   intArg(req.Arguments, "page_size", 8),
			HideVIP:    !hasMembership,
		})
		return newToolResult(canonicalName, data, err)
	case "list_membership_plans":
		data, err := s.service.ListPlans(ctx)
		return newToolResult(canonicalName, data, err)
	case "get_catalog_stats":
		data, err := s.service.Stats(ctx)
		return newToolResult(canonicalName, data, err)
	case "search_knowledge_base":
		data, err := s.service.SearchKnowledgeBase(ctx, domain.SearchKnowledgeBaseInput{
			SubjectKey:    stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Query:         firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Page:          intArg(req.Arguments, "page", 1),
			PageSize:      intArg(req.Arguments, "page_size", 10),
			LearnerUserID: session.UserID,
		})
		return knowledgeBaseToolResult(canonicalName, data, err)
	case "list_my_knowledge_base_documents":
		data, err := s.service.ListKnowledgeBaseDocuments(ctx, domain.KnowledgeBaseDocumentFilter{
			SubjectKey:         stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Query:              firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Page:               intArg(req.Arguments, "page", 1),
			PageSize:           intArg(req.Arguments, "page_size", 20),
			OnlyOwned:          true,
			OwnerLearnerUserID: session.UserID,
		})
		return newToolResult(canonicalName, data, err)
	case "view_knowledge_base_document":
		data, err := s.service.ListLearnerKnowledgeBaseDocumentChunks(ctx, session.UserID, uint(intArg(req.Arguments, "document_id", 0)), domain.KnowledgeBaseChunkFilter{
			Page:     intArg(req.Arguments, "page", 1),
			PageSize: intArg(req.Arguments, "page_size", 200),
		})
		return newToolResult(canonicalName, data, err)
	case "list_my_payment_orders":
		data, err := s.service.ListLearnerPaymentOrders(ctx, session.Username, domain.PaymentOrderFilter{
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Status:     stringArg(req.Arguments, "status", ""),
			Query:      firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Page:       intArg(req.Arguments, "page", 1),
			PageSize:   intArg(req.Arguments, "page_size", 10),
		})
		return newToolResult(canonicalName, data, err)
	case "list_my_memberships":
		data, err := s.service.ListLearnerMemberships(ctx, session.Username, domain.SubscriptionFilter{
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Status:     stringArg(req.Arguments, "status", ""),
			Query:      firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Page:       intArg(req.Arguments, "page", 1),
			PageSize:   intArg(req.Arguments, "page_size", 10),
		})
		return newToolResult(canonicalName, data, err)
	case "get_invite_summary":
		data, err := s.service.GetInviteSummary(ctx, session.UserID)
		return newToolResult(canonicalName, data, err)
	case "get_learning_summary":
		data, err := s.service.GetLearnerLearningSummary(ctx, session.UserID, stringArg(req.Arguments, "subject_key", session.SubjectKey))
		return newToolResult(canonicalName, data, err)
	case "list_learning_progress":
		data, err := s.service.ListLearnerWordProgress(ctx, session.UserID, domain.LearnerWordProgressFilter{
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Query:      firstNonEmpty(stringArg(req.Arguments, "query", ""), stringArg(req.Arguments, "q", "")),
			Level:      stringArg(req.Arguments, "level", ""),
			Difficulty: stringArg(req.Arguments, "difficulty", ""),
			Page:       intArg(req.Arguments, "page", 1),
			PageSize:   intArg(req.Arguments, "page_size", 20),
			DueOnly:    boolArg(req.Arguments, "due_only", false),
		})
		return newToolResult(canonicalName, data, err)
	case "save_learning_word_progress":
		data, err := s.service.SaveLearnerWordProgress(ctx, session.UserID, domain.SaveLearnerWordProgressInput{
			WordID:     uint64(intArg(req.Arguments, "word_id", 0)),
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Level:      stringArg(req.Arguments, "level", ""),
			Difficulty: stringArg(req.Arguments, "difficulty", ""),
		})
		return newToolResult(canonicalName, data, err)
	case "review_learning_word":
		data, err := s.service.ReviewLearnerWord(ctx, session.UserID, domain.ReviewLearnerWordInput{
			WordID:     uint64(intArg(req.Arguments, "word_id", 0)),
			SubjectKey: stringArg(req.Arguments, "subject_key", session.SubjectKey),
			Remembered: boolArg(req.Arguments, "remembered", false),
			Level:      stringArg(req.Arguments, "level", ""),
			Difficulty: stringArg(req.Arguments, "difficulty", ""),
		})
		return newToolResult(canonicalName, data, err)
	default:
		if result, handled, err := s.handleXiaomiTool(ctx, session, req); handled {
			return result, err
		}
		if result, handled, err := s.callDynamicAPITool(ctx, session, req); handled {
			return result, err
		}
		return CallToolResult{}, fmt.Errorf("unknown tool: %s", req.Name)
	}
}

func (s *Server) subjectMembershipAccess(ctx context.Context, session Session) (bool, error) {
	if strings.TrimSpace(session.Username) == "" {
		return false, nil
	}
	if strings.TrimSpace(session.SubjectKey) == "" {
		return s.service.LearnerHasAnyActiveMembership(ctx, session.Username)
	}
	return s.service.LearnerHasActiveMembership(ctx, session.Username, session.SubjectKey)
}

func newToolResult(toolName string, data interface{}, err error) (CallToolResult, error) {
	if err != nil {
		return CallToolResult{}, err
	}
	payload := map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"result":  data,
	}
	formatted, formatErr := json.MarshalIndent(payload, "", "  ")
	if formatErr != nil {
		return CallToolResult{}, formatErr
	}
	text := string(formatted)
	if optimized := formatToolText(toolName, data); optimized != "" {
		text = optimized
	}
	return CallToolResult{
		StructuredContent: payload,
		Content: []Content{
			{
				Type: "text",
				Text: text,
			},
		},
	}, nil
}

func formatToolText(toolName string, data interface{}) string {
	switch toolName {
	case "search_knowledge_base":
		result, ok := data.(domain.PagedKnowledgeBaseChunks)
		if !ok {
			return ""
		}
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("Knowledge base hits: %d\n", result.Total))
		for index, item := range result.Items {
			builder.WriteString(fmt.Sprintf("\n%d. %s", index+1, firstNonEmpty(item.DocumentTitle, item.Title, "Untitled Document")))
			if strings.TrimSpace(item.SourceFileName) != "" {
				builder.WriteString(fmt.Sprintf("\n   source: %s", item.SourceFileName))
			}
			builder.WriteString(fmt.Sprintf("\n   chunk: #%d", item.ChunkIndex))
			snippet := firstNonEmpty(item.HighlightedSnippet, item.Snippet, item.Content)
			if strings.TrimSpace(snippet) != "" {
				builder.WriteString(fmt.Sprintf("\n   snippet: %s\n", strings.TrimSpace(snippet)))
			} else {
				builder.WriteString("\n")
			}
		}
		return strings.TrimSpace(builder.String())
	default:
		return ""
	}
}

func knowledgeBaseToolResult(toolName string, data domain.PagedKnowledgeBaseChunks, err error) (CallToolResult, error) {
	if err != nil {
		return CallToolResult{}, err
	}
	hits := make([]map[string]interface{}, 0, len(data.Items))
	for _, item := range data.Items {
		hits = append(hits, map[string]interface{}{
			"document_id":         item.DocumentID,
			"document_title":      item.DocumentTitle,
			"source_file_name":    item.SourceFileName,
			"source_type":         item.SourceType,
			"source_label":        firstNonEmpty(item.SourceFileName, item.DocumentTitle, item.Title),
			"chunk_index":         item.ChunkIndex,
			"snippet":             item.Snippet,
			"highlighted_snippet": firstNonEmpty(item.HighlightedSnippet, item.Snippet),
			"status":              item.Status,
		})
	}
	payload := map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"result": map[string]interface{}{
			"items":     data.Items,
			"hits":      hits,
			"total":     data.Total,
			"page":      data.Page,
			"page_size": data.PageSize,
		},
	}
	text := formatToolText(toolName, data)
	if text == "" {
		formatted, formatErr := json.MarshalIndent(payload, "", "  ")
		if formatErr != nil {
			return CallToolResult{}, formatErr
		}
		text = string(formatted)
	}
	return CallToolResult{
		StructuredContent: payload,
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

func normalizeArguments(value interface{}) (map[string]interface{}, error) {
	switch typed := value.(type) {
	case nil:
		return map[string]interface{}{}, nil
	case map[string]interface{}:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return map[string]interface{}{}, nil
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("arguments must be an object or JSON object string")
		}
		if parsed == nil {
			return map[string]interface{}{}, nil
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("arguments must be an object")
	}
}

func canonicalToolName(name string) string {
	switch strings.TrimSpace(name) {
	case "list_subjects", "brights_list_subjects":
		return "list_subjects"
	case "list_categories", "brights_list_categories":
		return "list_categories"
	case "list_grades", "brights_list_grades":
		return "list_grades"
	case "list_words", "search_words", "brights_search_words":
		return "search_words"
	case "list_classification_stats", "brights_list_classification_stats":
		return "list_classification_stats"
	case "list_plans", "list_membership_plans", "brights_list_membership_plans":
		return "list_membership_plans"
	case "get_catalog_stats", "brights_get_catalog_stats":
		return "get_catalog_stats"
	case "search_knowledge_base", "brights_search_knowledge_base", "search_kb":
		return "search_knowledge_base"
	case "list_my_knowledge_base_documents", "my_knowledge_base_documents":
		return "list_my_knowledge_base_documents"
	case "view_knowledge_base_document", "knowledge_base_document_view":
		return "view_knowledge_base_document"
	case "list_my_payment_orders", "my_payment_orders", "brights_list_my_payment_orders":
		return "list_my_payment_orders"
	case "list_my_memberships", "my_memberships", "brights_list_my_memberships":
		return "list_my_memberships"
	case "get_invite_summary", "my_invite_summary", "brights_get_invite_summary":
		return "get_invite_summary"
	case "get_learning_summary", "my_learning_summary":
		return "get_learning_summary"
	case "list_learning_progress", "my_learning_progress":
		return "list_learning_progress"
	case "save_learning_word_progress", "update_learning_word_progress":
		return "save_learning_word_progress"
	case "review_learning_word", "remember_learning_word":
		return "review_learning_word"
	default:
		return strings.TrimSpace(name)
	}
}

func toolRequiresAuthenticatedSession(tool Tool) bool {
	if strings.EqualFold(strings.TrimSpace(tool.SourceType), "api_config_personal") {
		return true
	}
	switch canonicalToolName(tool.Name) {
	case "list_my_knowledge_base_documents", "view_knowledge_base_document",
		"list_my_payment_orders", "list_my_memberships", "get_invite_summary",
		"get_learning_summary", "list_learning_progress", "save_learning_word_progress", "review_learning_word",
		"xiaomi_get_devices", "xiaomi_extract_tokens", "xiaomi_miot_prop_get", "xiaomi_miot_prop_set",
		"xiaomi_miot_action", "xiaomi_miot_prop_get_batch", "xiaomi_find_device", "xiaomi_control_device",
		"list_mijia_homes", "get_mijia_devices", "get_device_status", "control_device", "get_device_spec",
		"mijia_list_devices", "mijia_get_caps", "mijia_switch_set", "mijia_sensor_get",
		"mijia_position_set", "mijia_action_call", "mijia_hvac_set":
		return true
	default:
		return false
	}
}

func newToolErrorResult(toolName string, err error) CallToolResult {
	payload := map[string]interface{}{
		"success": false,
		"tool":    canonicalToolName(toolName),
		"error":   err.Error(),
	}
	formatted, formatErr := json.MarshalIndent(payload, "", "  ")
	if formatErr != nil {
		formatted = []byte(fmt.Sprintf(`{"success":false,"tool":%q,"error":%q}`, canonicalToolName(toolName), err.Error()))
	}
	return CallToolResult{
		IsError:           true,
		StructuredContent: payload,
		Content: []Content{
			{
				Type: "text",
				Text: string(formatted),
			},
		},
	}
}

func isUnknownToolError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unknown tool")
}

func stringArg(args map[string]interface{}, key, fallback string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func intArg(args map[string]interface{}, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Server) toolMarket(ctx context.Context, session *Session, queryText, category string, page, pageSize int) ToolMarketResponse {
	page, pageSize = normalizeToolMarketPage(page, pageSize)
	tools := s.tools(ctx, session)
	categories := toolMarketCategories(tools)

	selectedCategory := strings.ToLower(strings.TrimSpace(category))
	searchText := strings.ToLower(strings.TrimSpace(queryText))
	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		if selectedCategory != "" && strings.ToLower(strings.TrimSpace(tool.Category)) != selectedCategory {
			continue
		}
		if searchText != "" {
			haystack := strings.ToLower(strings.Join([]string{
				tool.Title,
				tool.Name,
				tool.Description,
				tool.Category,
				tool.SourceType,
			}, " "))
			if !strings.Contains(haystack, searchText) {
				continue
			}
		}
		filtered = append(filtered, tool)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	items := make([]Tool, 0, end-start)
	if start < end {
		items = append(items, filtered[start:end]...)
	}

	return ToolMarketResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		Categories: categories,
	}
}

func normalizeToolMarketPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toolMarketCategories(tools []Tool) []string {
	seen := make(map[string]struct{}, len(tools))
	items := make([]string, 0, len(tools))
	for _, tool := range tools {
		category := strings.TrimSpace(tool.Category)
		if category == "" {
			category = "general"
		}
		key := strings.ToLower(category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, category)
	}
	sort.Strings(items)
	return items
}

func requestHTTPBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		scheme = forwarded
	}
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
