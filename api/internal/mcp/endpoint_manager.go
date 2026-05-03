package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"brights/api/internal/service"

	"github.com/gorilla/websocket"
)

const endpointHandshakeTimeout = 12 * time.Second

type EndpointConnectionManager struct {
	service *service.Service
	server  *Server

	mu      sync.RWMutex
	clients map[uint]*managedEndpointClient
}

type managedEndpointClient struct {
	endpointID uint
	learnerID  uint
	url        string

	mu          sync.Mutex
	conn        *websocket.Conn
	cancel      context.CancelFunc
	connectedAt *time.Time
	lastError   string
	status      string
}

type endpointConnectionSnapshot struct {
	Status      string
	IsConnected bool
	LastError   string
	ConnectedAt *time.Time
}

func NewEndpointConnectionManager(svc *service.Service, server *Server) *EndpointConnectionManager {
	return &EndpointConnectionManager{
		service: svc,
		server:  server,
		clients: make(map[uint]*managedEndpointClient),
	}
}

func (m *EndpointConnectionManager) RestoreAllEnabledEndpoints(ctx context.Context) error {
	endpoints, err := m.service.ListAllEnabledLearnerMCPEndpoints(ctx)
	if err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		if refreshErr := m.RefreshLearnerEndpoint(ctx, endpoint.ID, endpoint.LearnerUserID, endpoint.URL); refreshErr != nil {
			log.Printf("mcp endpoint manager: restore endpoint %d failed: %v", endpoint.ID, refreshErr)
		}
	}
	return nil
}

func (m *EndpointConnectionManager) RefreshLearnerEndpoint(ctx context.Context, endpointID uint, learnerID uint, endpointURL string) error {
	if learnerID == 0 || endpointID == 0 {
		return errors.New("learner id and endpoint id are required")
	}

	trimmedURL := strings.TrimSpace(endpointURL)
	if trimmedURL == "" {
		return errors.New("endpoint url is required")
	}

	client := m.ensureClient(endpointID, learnerID, trimmedURL)
	return client.connect(ctx, m.service, m.server)
}

func (m *EndpointConnectionManager) DisconnectEndpoint(ctx context.Context, endpointID uint, learnerID uint) error {
	if endpointID == 0 || learnerID == 0 {
		return errors.New("learner id and endpoint id are required")
	}

	m.mu.Lock()
	client, ok := m.clients[endpointID]
	if ok {
		delete(m.clients, endpointID)
	}
	m.mu.Unlock()

	if ok {
		client.closeWithStatus(ctx, m.service, "disconnected", "", nil)
		return nil
	}
	return m.service.UpdateLearnerMCPEndpointConnectionState(ctx, learnerID, endpointID, "disconnected", "", nil)
}

func (m *EndpointConnectionManager) RefreshLearnerEndpoints(ctx context.Context, learnerID uint) error {
	if learnerID == 0 {
		return errors.New("learner id is required")
	}

	endpoints, err := m.service.ListEnabledLearnerMCPEndpoints(ctx, learnerID)
	if err != nil {
		return err
	}

	activeIDs := make(map[uint]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		activeIDs[endpoint.ID] = struct{}{}
		if refreshErr := m.RefreshLearnerEndpoint(ctx, endpoint.ID, learnerID, endpoint.URL); refreshErr != nil {
			log.Printf("mcp endpoint manager: refresh endpoint %d failed: %v", endpoint.ID, refreshErr)
		}
	}

	m.mu.RLock()
	existingIDs := make([]uint, 0, len(m.clients))
	for endpointID, client := range m.clients {
		if client.learnerID == learnerID {
			existingIDs = append(existingIDs, endpointID)
		}
	}
	m.mu.RUnlock()

	for _, endpointID := range existingIDs {
		if _, ok := activeIDs[endpointID]; ok {
			continue
		}
		_ = m.DisconnectEndpoint(ctx, endpointID, learnerID)
	}

	return nil
}

func (m *EndpointConnectionManager) Snapshot(endpointID uint) endpointConnectionSnapshot {
	m.mu.RLock()
	client, ok := m.clients[endpointID]
	m.mu.RUnlock()
	if !ok {
		return endpointConnectionSnapshot{Status: "disconnected"}
	}
	return client.snapshot()
}

func (m *EndpointConnectionManager) ensureClient(endpointID uint, learnerID uint, endpointURL string) *managedEndpointClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[endpointID]
	if !ok {
		client = &managedEndpointClient{
			endpointID: endpointID,
			learnerID:  learnerID,
			url:        endpointURL,
			status:     "disconnected",
		}
		m.clients[endpointID] = client
		return client
	}

	client.mu.Lock()
	client.learnerID = learnerID
	client.url = endpointURL
	client.mu.Unlock()
	return client
}

func (c *managedEndpointClient) connect(ctx context.Context, svc *service.Service, server *Server) error {
	c.mu.Lock()
	cancel := c.cancel
	existingConn := c.conn
	c.cancel = nil
	c.conn = nil
	c.status = "connecting"
	c.lastError = ""
	c.connectedAt = nil
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if existingConn != nil {
		_ = existingConn.Close()
	}

	if err := svc.UpdateLearnerMCPEndpointConnectionState(ctx, c.learnerID, c.endpointID, "connecting", "", nil); err != nil {
		return err
	}

	dialCtx, cancel := context.WithTimeout(ctx, endpointHandshakeTimeout)
	defer cancel()

	dialer := websocket.Dialer{
		HandshakeTimeout: endpointHandshakeTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	conn, _, err := dialer.DialContext(dialCtx, c.url, nil)
	if err != nil {
		c.recordState("error", err.Error(), nil)
		_ = svc.UpdateLearnerMCPEndpointConnectionState(ctx, c.learnerID, c.endpointID, "error", err.Error(), nil)
		return err
	}

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	runCtx, runCancel := context.WithCancel(context.Background())
	connectedAt := time.Now().UTC()

	c.mu.Lock()
	c.conn = conn
	c.cancel = runCancel
	c.connectedAt = &connectedAt
	c.lastError = ""
	c.status = "connected"
	c.mu.Unlock()

	if err := svc.UpdateLearnerMCPEndpointConnectionState(ctx, c.learnerID, c.endpointID, "connected", "", &connectedAt); err != nil {
		runCancel()
		_ = conn.Close()
		return err
	}

	go c.readLoop(runCtx, conn, svc, server)
	go c.pingLoop(runCtx, conn, svc)
	return nil
}

func (c *managedEndpointClient) readLoop(ctx context.Context, conn *websocket.Conn, svc *service.Service, server *Server) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
				c.fail(conn, svc, err)
				return
			}
			_, payload, err := conn.ReadMessage()
			if err != nil {
				c.fail(conn, svc, err)
				return
			}
			if server == nil {
				continue
			}

			var req Request
			if err := json.Unmarshal(payload, &req); err != nil {
				if writeErr := c.writeResponse(conn, Response{
					JSONRPC: "2.0",
					Error: &Error{
						Code:    -32700,
						Message: "parse error",
						Data:    err.Error(),
					},
				}); writeErr != nil {
					c.fail(conn, svc, writeErr)
					return
				}
				continue
			}

			learner, learnerErr := svc.GetLearnerByID(context.Background(), c.learnerID)
			if learnerErr != nil {
				if writeErr := c.writeResponse(conn, Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &Error{
						Code:    -32000,
						Message: "learner not found",
						Data:    learnerErr.Error(),
					},
				}); writeErr != nil {
					c.fail(conn, svc, writeErr)
					return
				}
				continue
			}

			session := Session{
				UserID:   learner.ID,
				Username: learner.Username,
			}
			resp := server.handleRequest(context.Background(), session, req)
			if req.ID == nil && req.Method == "notifications/initialized" {
				continue
			}
			if err := c.writeResponse(conn, resp); err != nil {
				c.fail(conn, svc, err)
				return
			}
		}
	}
}

func (c *managedEndpointClient) pingLoop(ctx context.Context, conn *websocket.Conn, svc *service.Service) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.writeMessage(conn, websocket.PingMessage, nil); err != nil {
				c.fail(conn, svc, err)
				return
			}
		}
	}
}

func (c *managedEndpointClient) writeResponse(conn *websocket.Conn, resp Response) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn == nil {
		return errors.New("connection is closed")
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteJSON(resp)
}

func (c *managedEndpointClient) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn == nil {
		return errors.New("connection is closed")
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(messageType, data)
}

func (c *managedEndpointClient) fail(conn *websocket.Conn, svc *service.Service, err error) {
	c.mu.Lock()
	isCurrentConn := c.conn == conn
	c.mu.Unlock()
	if !isCurrentConn {
		return
	}

	if isExpectedSocketClose(err) {
		c.closeWithStatus(context.Background(), svc, "disconnected", "", nil)
		return
	}

	log.Printf("mcp endpoint manager: endpoint %d disconnected: %v", c.endpointID, err)
	c.closeWithStatus(context.Background(), svc, "error", err.Error(), nil)
}

func (c *managedEndpointClient) closeWithStatus(
	ctx context.Context,
	svc *service.Service,
	status string,
	lastError string,
	connectedAt *time.Time,
) {
	c.mu.Lock()
	cancel := c.cancel
	conn := c.conn
	c.cancel = nil
	c.conn = nil
	c.status = status
	c.lastError = strings.TrimSpace(lastError)
	c.connectedAt = connectedAt
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}

	if err := svc.UpdateLearnerMCPEndpointConnectionState(ctx, c.learnerID, c.endpointID, status, lastError, connectedAt); err != nil {
		log.Printf("mcp endpoint manager: update endpoint %d state failed: %v", c.endpointID, err)
	}
}

func (c *managedEndpointClient) recordState(status string, lastError string, connectedAt *time.Time) {
	c.mu.Lock()
	c.status = status
	c.lastError = strings.TrimSpace(lastError)
	c.connectedAt = connectedAt
	c.mu.Unlock()
}

func (c *managedEndpointClient) snapshot() endpointConnectionSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := strings.TrimSpace(c.status)
	if status == "" {
		status = "disconnected"
	}
	return endpointConnectionSnapshot{
		Status:      status,
		IsConnected: status == "connected",
		LastError:   c.lastError,
		ConnectedAt: c.connectedAt,
	}
}

func isExpectedSocketClose(err error) bool {
	if err == nil {
		return false
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return true
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "use of closed network connection") ||
		strings.Contains(message, "websocket: close sent")
}
