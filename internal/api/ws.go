package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/aezell/agrev/internal/analysis"
	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local dev; restrict in production
	},
}

// WebSocket message types from client.
const (
	wsMsgLoadDiff = "load_diff"
	wsMsgApprove  = "approve"
	wsMsgReject   = "reject"
	wsMsgUndo     = "undo"
	wsMsgFinish   = "finish"
)

// WebSocket message types to client.
const (
	wsMsgParsed   = "parsed"
	wsMsgAnalysis = "analysis"
	wsMsgDecision = "decision"
	wsMsgSummary  = "summary"
	wsMsgError    = "error"
)

// wsMessage is the envelope for WebSocket messages in both directions.
type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// wsLoadDiff is the payload for "load_diff" messages.
type wsLoadDiff struct {
	Diff    string   `json:"diff"`
	RepoDir string   `json:"repo_dir,omitempty"`
	Skip    []string `json:"skip,omitempty"`
}

// wsDecisionMsg is the payload for approve/reject/undo messages.
type wsDecisionMsg struct {
	FileIndex int `json:"file_index"`
}

// wsParsedResponse is sent after a diff is loaded.
type wsParsedResponse struct {
	Files []fileJSON    `json:"files"`
	Stats diffStatsJSON `json:"stats"`
}

// wsAnalysisResponse is sent after analysis completes.
type wsAnalysisResponse struct {
	Summary  string        `json:"summary"`
	MaxRisk  string        `json:"max_risk"`
	Total    int           `json:"total"`
	Findings []findingJSON `json:"findings"`
}

// wsDecisionResponse confirms a decision.
type wsDecisionResponse struct {
	FileIndex int    `json:"file_index"`
	Decision  string `json:"decision"`
}

// wsSummaryResponse is sent when the review is finished.
type wsSummaryResponse struct {
	Approved int      `json:"approved"`
	Rejected int      `json:"rejected"`
	Pending  int      `json:"pending"`
	Files    []wsFileDecision `json:"files"`
}

type wsFileDecision struct {
	Name     string `json:"name"`
	Decision string `json:"decision"`
}

// reviewSession holds the state for a WebSocket review session.
type reviewSession struct {
	ds        *diff.DiffSet
	results   *analysis.Results
	decisions map[int]model.ReviewDecision
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	session := &reviewSession{
		decisions: make(map[int]model.ReviewDecision),
	}

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket read: %v", err)
			}
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			sendWSError(conn, "invalid message format")
			continue
		}

		switch msg.Type {
		case wsMsgLoadDiff:
			handleWSLoadDiff(conn, session, msg.Data)
		case wsMsgApprove:
			handleWSDecision(conn, session, msg.Data, model.DecisionApproved)
		case wsMsgReject:
			handleWSDecision(conn, session, msg.Data, model.DecisionRejected)
		case wsMsgUndo:
			handleWSUndo(conn, session, msg.Data)
		case wsMsgFinish:
			handleWSFinish(conn, session)
		default:
			sendWSError(conn, "unknown message type: "+msg.Type)
		}
	}
}

func handleWSLoadDiff(conn *websocket.Conn, session *reviewSession, data json.RawMessage) {
	var req wsLoadDiff
	if err := json.Unmarshal(data, &req); err != nil {
		sendWSError(conn, "invalid load_diff data")
		return
	}

	ds, err := diff.Parse(req.Diff)
	if err != nil {
		sendWSError(conn, "parsing diff: "+err.Error())
		return
	}

	session.ds = ds
	session.decisions = make(map[int]model.ReviewDecision)

	// Send parsed response
	nFiles, added, deleted := ds.Stats()
	parsed := wsParsedResponse{
		Stats: diffStatsJSON{Files: nFiles, Added: added, Deleted: deleted},
	}
	for _, f := range ds.Files {
		parsed.Files = append(parsed.Files, fileJSON{
			Name:         f.Name(),
			OldName:      f.OldName,
			NewName:      f.NewName,
			IsNew:        f.IsNew,
			IsDeleted:    f.IsDeleted,
			IsRenamed:    f.IsRenamed,
			AddedLines:   f.AddedLines,
			DeletedLines: f.DeletedLines,
			Fragments:    len(f.Fragments),
		})
	}
	sendWSMessage(conn, wsMsgParsed, parsed)

	// Run analysis
	results := analysis.Run(ds, req.RepoDir, req.Skip)
	session.results = results

	analysisResp := wsAnalysisResponse{
		Summary: results.Summary(),
		MaxRisk: results.MaxRisk().String(),
		Total:   len(results.Findings),
	}
	for _, f := range results.Findings {
		analysisResp.Findings = append(analysisResp.Findings, findingJSON{
			Pass:     f.Pass,
			File:     f.File,
			Line:     f.Line,
			Message:  f.Message,
			Severity: severityStr(f.Severity),
			Risk:     f.Risk.String(),
		})
	}
	sendWSMessage(conn, wsMsgAnalysis, analysisResp)
}

func handleWSDecision(conn *websocket.Conn, session *reviewSession, data json.RawMessage, decision model.ReviewDecision) {
	if session.ds == nil {
		sendWSError(conn, "no diff loaded")
		return
	}

	var req wsDecisionMsg
	if err := json.Unmarshal(data, &req); err != nil {
		sendWSError(conn, "invalid decision data")
		return
	}

	if req.FileIndex < 0 || req.FileIndex >= len(session.ds.Files) {
		sendWSError(conn, "file_index out of range")
		return
	}

	session.decisions[req.FileIndex] = decision

	decisionStr := "approved"
	if decision == model.DecisionRejected {
		decisionStr = "rejected"
	}

	sendWSMessage(conn, wsMsgDecision, wsDecisionResponse{
		FileIndex: req.FileIndex,
		Decision:  decisionStr,
	})
}

func handleWSUndo(conn *websocket.Conn, session *reviewSession, data json.RawMessage) {
	if session.ds == nil {
		sendWSError(conn, "no diff loaded")
		return
	}

	var req wsDecisionMsg
	if err := json.Unmarshal(data, &req); err != nil {
		sendWSError(conn, "invalid undo data")
		return
	}

	delete(session.decisions, req.FileIndex)

	sendWSMessage(conn, wsMsgDecision, wsDecisionResponse{
		FileIndex: req.FileIndex,
		Decision:  "pending",
	})
}

func handleWSFinish(conn *websocket.Conn, session *reviewSession) {
	if session.ds == nil {
		sendWSError(conn, "no diff loaded")
		return
	}

	var approved, rejected, pending int
	var files []wsFileDecision

	for i, f := range session.ds.Files {
		fd := wsFileDecision{Name: f.Name()}
		switch session.decisions[i] {
		case model.DecisionApproved:
			fd.Decision = "approved"
			approved++
		case model.DecisionRejected:
			fd.Decision = "rejected"
			rejected++
		default:
			fd.Decision = "pending"
			pending++
		}
		files = append(files, fd)
	}

	sendWSMessage(conn, wsMsgSummary, wsSummaryResponse{
		Approved: approved,
		Rejected: rejected,
		Pending:  pending,
		Files:    files,
	})
}

func sendWSMessage(conn *websocket.Conn, msgType string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf("ws marshal: %v", err)
		return
	}
	msg := wsMessage{Type: msgType, Data: raw}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("ws write: %v", err)
	}
}

func sendWSError(conn *websocket.Conn, errMsg string) {
	sendWSMessage(conn, wsMsgError, map[string]string{"message": errMsg})
}
