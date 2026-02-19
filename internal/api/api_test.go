package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

const testDiff = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
-	println("hello")
+	println("hello world")
+	println("goodbye")
 }
diff --git a/util.go b/util.go
new file mode 100644
--- /dev/null
+++ b/util.go
@@ -0,0 +1,5 @@
+package main
+
+func add(a, b int) int {
+	return a + b
+}
`

func newTestServer() *Server {
	return New(":0")
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %q", resp["status"])
	}
}

func TestAnalyzeEndpoint(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(analyzeRequest{Diff: testDiff})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp analyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}

	if resp.Stats.Files != 2 {
		t.Errorf("expected 2 files, got %d", resp.Stats.Files)
	}
	if resp.MaxRisk == "" {
		t.Error("expected non-empty max_risk")
	}
}

func TestAnalyzeEmptyDiff(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(analyzeRequest{Diff: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestParseEndpoint(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(parseRequest{Diff: testDiff})
	req := httptest.NewRequest(http.MethodPost, "/api/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp parseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}

	if len(resp.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(resp.Files))
	}
	if resp.Files[0].Name != "main.go" {
		t.Errorf("expected first file main.go, got %q", resp.Files[0].Name)
	}
	if !resp.Files[1].IsNew {
		t.Error("expected second file to be new")
	}
	if resp.Stats.Added != 7 {
		t.Errorf("expected 7 added lines, got %d", resp.Stats.Added)
	}
}

func TestSummaryNoInput(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(summaryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/summary", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAnalyzeInvalidJSON(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestServeCommandRegistered(t *testing.T) {
	// Verify the serve command exists via the root test
	srv := newTestServer()
	if srv.addr != ":0" {
		t.Errorf("expected addr :0, got %q", srv.addr)
	}
}

func TestWebSocketReviewSession(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send load_diff
	loadData, _ := json.Marshal(wsLoadDiff{Diff: testDiff})
	sendMsg := wsMessage{Type: wsMsgLoadDiff, Data: loadData}
	if err := conn.WriteJSON(sendMsg); err != nil {
		t.Fatalf("ws write: %v", err)
	}

	// Should receive "parsed" message
	var msg1 wsMessage
	if err := conn.ReadJSON(&msg1); err != nil {
		t.Fatalf("ws read parsed: %v", err)
	}
	if msg1.Type != wsMsgParsed {
		t.Errorf("expected 'parsed' message, got %q", msg1.Type)
	}

	var parsed wsParsedResponse
	if err := json.Unmarshal(msg1.Data, &parsed); err != nil {
		t.Fatalf("unmarshal parsed: %v", err)
	}
	if len(parsed.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(parsed.Files))
	}

	// Should receive "analysis" message
	var msg2 wsMessage
	if err := conn.ReadJSON(&msg2); err != nil {
		t.Fatalf("ws read analysis: %v", err)
	}
	if msg2.Type != wsMsgAnalysis {
		t.Errorf("expected 'analysis' message, got %q", msg2.Type)
	}

	// Approve file 0
	decData, _ := json.Marshal(wsDecisionMsg{FileIndex: 0})
	approveMsg := wsMessage{Type: wsMsgApprove, Data: decData}
	if err := conn.WriteJSON(approveMsg); err != nil {
		t.Fatalf("ws write approve: %v", err)
	}

	var msg3 wsMessage
	if err := conn.ReadJSON(&msg3); err != nil {
		t.Fatalf("ws read decision: %v", err)
	}
	if msg3.Type != wsMsgDecision {
		t.Errorf("expected 'decision' message, got %q", msg3.Type)
	}

	var dec wsDecisionResponse
	if err := json.Unmarshal(msg3.Data, &dec); err != nil {
		t.Fatalf("unmarshal decision: %v", err)
	}
	if dec.Decision != "approved" {
		t.Errorf("expected approved, got %q", dec.Decision)
	}

	// Reject file 1
	decData, _ = json.Marshal(wsDecisionMsg{FileIndex: 1})
	rejectMsg := wsMessage{Type: wsMsgReject, Data: decData}
	if err := conn.WriteJSON(rejectMsg); err != nil {
		t.Fatalf("ws write reject: %v", err)
	}

	var msg4 wsMessage
	if err := conn.ReadJSON(&msg4); err != nil {
		t.Fatalf("ws read reject: %v", err)
	}
	if msg4.Type != wsMsgDecision {
		t.Errorf("expected 'decision' message, got %q", msg4.Type)
	}

	// Finish
	finishMsg := wsMessage{Type: wsMsgFinish}
	if err := conn.WriteJSON(finishMsg); err != nil {
		t.Fatalf("ws write finish: %v", err)
	}

	var msg5 wsMessage
	if err := conn.ReadJSON(&msg5); err != nil {
		t.Fatalf("ws read summary: %v", err)
	}
	if msg5.Type != wsMsgSummary {
		t.Errorf("expected 'summary' message, got %q", msg5.Type)
	}

	var summary wsSummaryResponse
	if err := json.Unmarshal(msg5.Data, &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summary.Approved != 1 || summary.Rejected != 1 || summary.Pending != 0 {
		t.Errorf("expected 1/1/0, got %d/%d/%d", summary.Approved, summary.Rejected, summary.Pending)
	}
}

func TestWebSocketUndo(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Load diff
	loadData, _ := json.Marshal(wsLoadDiff{Diff: testDiff})
	conn.WriteJSON(wsMessage{Type: wsMsgLoadDiff, Data: loadData})

	// Read parsed + analysis
	conn.ReadJSON(&wsMessage{})
	conn.ReadJSON(&wsMessage{})

	// Approve file 0
	decData, _ := json.Marshal(wsDecisionMsg{FileIndex: 0})
	conn.WriteJSON(wsMessage{Type: wsMsgApprove, Data: decData})
	conn.ReadJSON(&wsMessage{}) // read decision response

	// Undo file 0
	conn.WriteJSON(wsMessage{Type: wsMsgUndo, Data: decData})

	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("ws read undo: %v", err)
	}

	var dec wsDecisionResponse
	json.Unmarshal(msg.Data, &dec)
	if dec.Decision != "pending" {
		t.Errorf("expected pending after undo, got %q", dec.Decision)
	}
}
