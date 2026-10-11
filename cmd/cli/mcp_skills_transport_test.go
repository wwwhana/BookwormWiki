package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wwwhana/bookwormwiki/internal/auth"
	"github.com/wwwhana/bookwormwiki/internal/bootstrap"
)

func TestBookwormWikiSkillsStreamableHTTPEndToEnd(t *testing.T) {
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{})

	initialize := postBookwormWikiMCP(t, handler, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	assertBookwormWikiMCPResponse(t, initialize, http.StatusOK, "no-store")
	sessionID := initialize.Header().Get(server.HeaderKeySessionID)
	if sessionID == "" {
		t.Fatal("initialize response did not include Mcp-Session-Id")
	}
	initialized := decodeBookwormWikiHTTPResult[mcp.InitializeResult](t, initialize)
	extension, ok := initialized.Capabilities.Extensions[skillsExtensionName].(map[string]any)
	if !ok || extension["directoryRead"] != true {
		t.Fatalf("initialize skills extension = %#v", initialized.Capabilities.Extensions)
	}
	if initialized.Capabilities.Resources == nil {
		t.Fatal("initialize did not advertise standard resources")
	}

	notification := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`)
	assertBookwormWikiMCPResponse(t, notification, http.StatusAccepted, "no-store")
	if notification.Body.Len() != 0 {
		t.Fatalf("initialized notification body = %q, want empty", notification.Body.String())
	}

	listResponse := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":2,"method":"skills/list","params":{}}`)
	assertBookwormWikiMCPResponse(t, listResponse, http.StatusOK, "no-store")
	list := decodeBookwormWikiHTTPResult[skillsListResult](t, listResponse)
	if list.ResultType != "complete" || len(list.Skills) != 2 || list.Skills[1].URI != bookwormwikiWorkSkillURI || list.Skills[0].URI != "skill://bookwormwiki-wiki/SKILL.md" {
		t.Fatalf("skills/list result = %#v", list)
	}

	getResponse := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":3,"method":"skills/get","params":{"uri":"skill://bookwormwiki-work/SKILL.md"}}`)
	assertBookwormWikiMCPResponse(t, getResponse, http.StatusOK, "no-store")
	get := decodeBookwormWikiHTTPResult[skillsGetResult](t, getResponse)
	if get.ResultType != "complete" || get.Skill.URI != bookwormwikiWorkSkillURI {
		t.Fatalf("skills/get result = %#v", get)
	}

	directoryResponse := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":4,"method":"resources/directory/read","params":{"uri":"skill://bookwormwiki-work"}}`)
	assertBookwormWikiMCPResponse(t, directoryResponse, http.StatusOK, "no-store")
	directory := decodeBookwormWikiHTTPResult[directoryReadResult](t, directoryResponse)
	if directory.ResultType != "complete" || len(directory.Resources) == 0 {
		t.Fatalf("resources/directory/read result = %#v", directory)
	}
	assertResource(t, directory.Resources, bookwormwikiWorkSkillURI, "bookwormwiki-work", "text/markdown")

	readResponse := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"skill://bookwormwiki-work/SKILL.md"}}`)
	assertBookwormWikiMCPResponse(t, readResponse, http.StatusOK, "no-store")
	read := decodeBookwormWikiHTTPResult[struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}](t, readResponse)
	if len(read.Contents) != 1 || read.Contents[0].URI != bookwormwikiWorkSkillURI || read.Contents[0].MIMEType != "text/markdown" || !strings.Contains(read.Contents[0].Text, "name: bookwormwiki-work") {
		t.Fatalf("resources/read result = %#v", read)
	}
}

func TestBookwormWikiSkillsHTTPTransportAnswersHeadProbe(t *testing.T) {
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/mcp", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("HEAD /mcp status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("HEAD /mcp body = %q, want empty", response.Body.String())
	}
	if got := response.Header().Get("Allow"); got != "GET, POST, DELETE" {
		t.Fatalf("HEAD /mcp Allow = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("HEAD /mcp Content-Type = %q", got)
	}
}

func TestBookwormWikiSkillsHTTPTransportAnswersHeaderlessGetProbe(t *testing.T) {
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /mcp probe status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("GET /mcp probe body = %q, want empty", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("GET /mcp probe Content-Type = %q", got)
	}
}

func TestBookwormWikiSkillsHTTPTransportKeepsProbesBehindAuthentication(t *testing.T) {
	provider, err := auth.Init(context.Background(), auth.Config{Mode: "token", APISecret: testAuthSecret})
	if err != nil {
		t.Fatalf("init token auth: %v", err)
	}
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{Auth: provider})
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/mcp", nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s /mcp probe status = %d, want %d", method, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestBookwormWikiSkillsAreNotAdvertisedWithoutHTTPDispatcher(t *testing.T) {
	mcpServer := newMCPServer(nil)
	initialize := json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	result := rpcResult[mcp.InitializeResult](t, mcpServer.HandleMessage(t.Context(), initialize))
	if _, advertised := result.Capabilities.Extensions[skillsExtensionName]; advertised {
		t.Fatalf("unreachable skills methods were advertised: %#v", result.Capabilities.Extensions)
	}
	if result.Capabilities.Resources == nil {
		t.Fatal("standard resources were not registered on the base MCP server")
	}

	response := mcpServer.HandleMessage(t.Context(), json.RawMessage(`{"jsonrpc":"2.0","id":2,"method":"skills/list","params":{}}`))
	rpcError, ok := response.(mcp.JSONRPCError)
	if !ok || rpcError.Error.Code != mcp.METHOD_NOT_FOUND {
		t.Fatalf("unwrapped skills/list response = %#v, want method not found", response)
	}
}

func TestBookwormWikiSkillsHTTPTransportUsesAuthenticatedContextAndForwardsBatches(t *testing.T) {
	var handledMode string
	var handledWithTransport bool
	handledCount := 0
	var forwardedBody string
	var forwardedWithTransport bool

	transport := &bookwormwikiSkillsHTTPTransport{
		next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			forwardedBody = string(body)
			forwardedWithTransport = bookwormwikiSkillsTransportAvailable(r.Context())
			w.WriteHeader(http.StatusAccepted)
		}),
		sessionResolver: server.NewDefaultSessionIdManagerResolver(&server.StatelessSessionIdManager{}),
		handleMessage: func(ctx context.Context, _ json.RawMessage) (mcp.JSONRPCMessage, bool) {
			handledCount++
			handledMode, _ = ctx.Value(keyMode).(string)
			handledWithTransport = bookwormwikiSkillsTransportAvailable(ctx)
			return mcp.NewJSONRPCResultResponse(mcp.NewRequestId(1), map[string]any{"ok": true}), true
		},
	}
	handler := authenticatedHTTP(nil, transport)

	custom := postBookwormWikiMCP(t, handler, "", `{"jsonrpc":"2.0","id":1,"method":"skills/list","params":{}}`)
	assertBookwormWikiMCPResponse(t, custom, http.StatusOK, "no-store")
	if handledMode != "local" || !handledWithTransport {
		t.Fatalf("custom handler context mode=%q transport=%v", handledMode, handledWithTransport)
	}

	batchBody := `[{"jsonrpc":"2.0","id":2,"method":"skills/list","params":{}},{"jsonrpc":"2.0","id":3,"method":"ping"}]`
	batch := postBookwormWikiMCP(t, handler, "", batchBody)
	assertBookwormWikiMCPResponse(t, batch, http.StatusAccepted, "no-store")
	if handledCount != 1 {
		t.Fatalf("batch invoked custom handler; handled count = %d", handledCount)
	}
	if forwardedBody != batchBody || !forwardedWithTransport {
		t.Fatalf("forwarded batch body=%q transport=%v", forwardedBody, forwardedWithTransport)
	}
}

func TestBookwormWikiSkillsHTTPTransportReturnsSessionAndJSONRPCErrors(t *testing.T) {
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{})

	missingSession := postBookwormWikiMCP(t, handler, "", `{"jsonrpc":"2.0","id":1,"method":"skills/list","params":{}}`)
	assertBookwormWikiMCPResponse(t, missingSession, http.StatusNotFound, "no-store")
	if !strings.Contains(missingSession.Body.String(), "Invalid session ID") {
		t.Fatalf("missing-session body = %q", missingSession.Body.String())
	}

	initialize := postBookwormWikiMCP(t, handler, "", `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	sessionID := initialize.Header().Get(server.HeaderKeySessionID)
	if sessionID == "" {
		t.Fatal("initialize response did not include Mcp-Session-Id")
	}

	invalidParams := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":"bad","method":"skills/get","params":{"uri":42}}`)
	assertBookwormWikiMCPResponse(t, invalidParams, http.StatusOK, "no-store")
	if rpcError := decodeBookwormWikiHTTPError(t, invalidParams); rpcError.Code != mcp.INVALID_PARAMS {
		t.Fatalf("invalid params code = %d, want %d", rpcError.Code, mcp.INVALID_PARAMS)
	}

	invalidRequest := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"1.0","id":3,"method":"skills/list","params":{}}`)
	assertBookwormWikiMCPResponse(t, invalidRequest, http.StatusOK, "no-store")
	if rpcError := decodeBookwormWikiHTTPError(t, invalidRequest); rpcError.Code != mcp.INVALID_REQUEST {
		t.Fatalf("invalid request code = %d, want %d", rpcError.Code, mcp.INVALID_REQUEST)
	}

	notification := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","method":"skills/list","params":{}}`)
	assertBookwormWikiMCPResponse(t, notification, http.StatusAccepted, "no-store")
	if notification.Body.Len() != 0 {
		t.Fatalf("skills/list notification body = %q, want empty", notification.Body.String())
	}
}

func TestBookwormWikiSkillsHTTPTransportRejectsOversizedRequests(t *testing.T) {
	handler := newBookwormWikiHTTPHandler(&bootstrap.Context{})
	body := `{"jsonrpc":"2.0","id":1,"method":"skills/list","params":{"cursor":"` + strings.Repeat("a", maxMCPRequestBodyBytes) + `"}}`
	response := postBookwormWikiMCP(t, handler, "mcp-session-00000000-0000-0000-0000-000000000000", body)
	assertBookwormWikiMCPResponse(t, response, http.StatusBadRequest, "no-store")
	if !strings.Contains(response.Body.String(), "request body too large") {
		t.Fatalf("oversized request body = %q", response.Body.String())
	}
}

func TestBookwormWikiSkillsHTTPTransportPreservesStreamableSSEPostResponse(t *testing.T) {
	protocol, _ := testBookwormWikiSkillsProtocol(t, defaultSkillPageSize, defaultDirectoryPageSize)
	mcpServer := server.NewMCPServer("test", "test")
	protocol.Register(mcpServer)
	mcpServer.GetHooks().AddBeforePing(func(ctx context.Context, _ any, _ *mcp.PingRequest) {
		session := server.ClientSessionFromContext(ctx)
		if session == nil {
			t.Error("ping did not receive Streamable HTTP session context")
			return
		}
		streamableSession, ok := session.(server.SessionWithStreamableHTTPConfig)
		if !ok {
			t.Errorf("ping session type = %T, want Streamable HTTP session", session)
			return
		}
		streamableSession.UpgradeToSSEWhenReceiveNotification()
		session.NotificationChannel() <- mcp.JSONRPCNotification{
			JSONRPC: mcp.JSONRPC_VERSION,
			Notification: mcp.Notification{
				Method: "notifications/test",
			},
		}
	})

	sessionResolver := server.NewDefaultSessionIdManagerResolver(&server.StatelessGeneratingSessionIdManager{})
	streamable := server.NewStreamableHTTPServer(mcpServer,
		server.WithHTTPContextFunc(httpContextFunc),
		server.WithSessionIdManagerResolver(sessionResolver),
	)
	handler := authenticatedHTTP(nil, newBookwormWikiSkillsHTTPTransport(streamable, sessionResolver))

	initialize := postBookwormWikiMCP(t, handler, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	sessionID := initialize.Header().Get(server.HeaderKeySessionID)
	if sessionID == "" {
		t.Fatal("initialize response did not include Mcp-Session-Id")
	}

	ping := postBookwormWikiMCP(t, handler, sessionID, `{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`)
	if ping.Code != http.StatusOK {
		t.Fatalf("ping status = %d, want %d; body=%s", ping.Code, http.StatusOK, ping.Body.String())
	}
	if contentType := ping.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("ping content type = %q, want text/event-stream", contentType)
	}
	if cacheControl := ping.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("SSE cache control = %q, want no-store", cacheControl)
	}
	if body := ping.Body.String(); !strings.Contains(body, "event: message") || !strings.Contains(body, `"id":2`) {
		t.Fatalf("SSE ping body = %q", body)
	}
}

func postBookwormWikiMCP(t *testing.T, handler http.Handler, sessionID, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		request.Header.Set(server.HeaderKeySessionID, sessionID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertBookwormWikiMCPResponse(t *testing.T, response *httptest.ResponseRecorder, status int, cacheControl string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("HTTP status = %d, want %d; body=%s", response.Code, status, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != cacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, cacheControl)
	}
}

func decodeBookwormWikiHTTPResult[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var envelope struct {
		Result json.RawMessage          `json:"result"`
		Error  *mcp.JSONRPCErrorDetails `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode JSON-RPC response: %v; body=%s", err, response.Body.String())
	}
	if envelope.Error != nil {
		t.Fatalf("JSON-RPC response returned error: %#v", envelope.Error)
	}
	var result T
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode JSON-RPC result: %v; body=%s", err, response.Body.String())
	}
	return result
}

func decodeBookwormWikiHTTPError(t *testing.T, response *httptest.ResponseRecorder) mcp.JSONRPCErrorDetails {
	t.Helper()
	var envelope struct {
		Error mcp.JSONRPCErrorDetails `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode JSON-RPC error: %v; body=%s", err, response.Body.String())
	}
	return envelope.Error
}
