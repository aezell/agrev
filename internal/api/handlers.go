package api

import (
	"net/http"

	"github.com/aezell/agrev/internal/analysis"
	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
	"github.com/aezell/agrev/internal/trace"
)

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Analyze ---

type analyzeRequest struct {
	Diff    string   `json:"diff"`
	RepoDir string   `json:"repo_dir,omitempty"`
	Skip    []string `json:"skip,omitempty"`
}

type analyzeResponse struct {
	Summary  string           `json:"summary"`
	MaxRisk  string           `json:"max_risk"`
	Total    int              `json:"total"`
	Findings []findingJSON    `json:"findings"`
	Stats    diffStatsJSON    `json:"stats"`
}

type findingJSON struct {
	Pass     string `json:"pass"`
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Risk     string `json:"risk"`
}

type diffStatsJSON struct {
	Files   int `json:"files"`
	Added   int `json:"added"`
	Deleted int `json:"deleted"`
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Diff == "" {
		writeError(w, http.StatusBadRequest, "diff is required")
		return
	}

	ds, err := diff.Parse(req.Diff)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parsing diff: "+err.Error())
		return
	}

	results := analysis.Run(ds, req.RepoDir, req.Skip)

	nFiles, added, deleted := ds.Stats()
	resp := analyzeResponse{
		Summary: results.Summary(),
		MaxRisk: results.MaxRisk().String(),
		Total:   len(results.Findings),
		Stats: diffStatsJSON{
			Files:   nFiles,
			Added:   added,
			Deleted: deleted,
		},
	}

	for _, f := range results.Findings {
		resp.Findings = append(resp.Findings, findingJSON{
			Pass:     f.Pass,
			File:     f.File,
			Line:     f.Line,
			Message:  f.Message,
			Severity: severityStr(f.Severity),
			Risk:     f.Risk.String(),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Parse ---

type parseRequest struct {
	Diff string `json:"diff"`
}

type parseResponse struct {
	Files []fileJSON    `json:"files"`
	Stats diffStatsJSON `json:"stats"`
}

type fileJSON struct {
	Name         string `json:"name"`
	OldName      string `json:"old_name,omitempty"`
	NewName      string `json:"new_name,omitempty"`
	IsNew        bool   `json:"is_new,omitempty"`
	IsDeleted    bool   `json:"is_deleted,omitempty"`
	IsRenamed    bool   `json:"is_renamed,omitempty"`
	AddedLines   int    `json:"added_lines"`
	DeletedLines int    `json:"deleted_lines"`
	Fragments    int    `json:"fragments"`
}

func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	var req parseRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Diff == "" {
		writeError(w, http.StatusBadRequest, "diff is required")
		return
	}

	ds, err := diff.Parse(req.Diff)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parsing diff: "+err.Error())
		return
	}

	nFiles, added, deleted := ds.Stats()
	resp := parseResponse{
		Stats: diffStatsJSON{
			Files:   nFiles,
			Added:   added,
			Deleted: deleted,
		},
	}

	for _, f := range ds.Files {
		resp.Files = append(resp.Files, fileJSON{
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

	writeJSON(w, http.StatusOK, resp)
}

// --- Summary ---

type summaryRequest struct {
	TracePath string `json:"trace_path"`
	RepoDir   string `json:"repo_dir,omitempty"`
}

type summaryResponse struct {
	Source       string   `json:"source"`
	Summary      string   `json:"summary"`
	Steps        int      `json:"steps"`
	FilesChanged []string `json:"files_changed"`
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	var req summaryRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	var t *trace.Trace
	var err error

	if req.TracePath != "" {
		t, err = trace.Load(req.TracePath, "")
		if err != nil {
			writeError(w, http.StatusBadRequest, "loading trace: "+err.Error())
			return
		}
	} else if req.RepoDir != "" {
		t, err = trace.DetectAndLoad(req.RepoDir)
		if err != nil {
			writeError(w, http.StatusBadRequest, "detecting trace: "+err.Error())
			return
		}
	} else {
		writeError(w, http.StatusBadRequest, "trace_path or repo_dir is required")
		return
	}

	if t == nil {
		writeError(w, http.StatusNotFound, "no trace found")
		return
	}

	resp := summaryResponse{
		Source:       t.Source,
		Summary:      t.Summary,
		Steps:        len(t.Steps),
		FilesChanged: t.FilesChanged,
	}

	writeJSON(w, http.StatusOK, resp)
}

func severityStr(s model.Severity) string {
	switch s {
	case model.SeverityError:
		return "error"
	case model.SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}
