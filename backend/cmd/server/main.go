// Command server starts the HTTP API for MIDI beat generation.
//
// Endpoints:
//   GET  /api/info       --style info, BPM ranges, max bars per tier
//   POST /api/generate   --generate a MIDI beat (free tier: max 8 bars)
//
// Environment:
//   PORT              --listen port (default 8080)
//   OPENAI_API_KEY    --required for LLM calls
//   OPENAI_MODEL      --model name (default deepseek-chat)
//   OPENAI_BASE_URL   --API base URL (default https://api.deepseek.com/v1)
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/ShowerBandV/text2midi/internal/agent"
	"github.com/ShowerBandV/text2midi/internal/composer"
	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/schema"
	"github.com/ShowerBandV/text2midi/internal/store"
	"github.com/ShowerBandV/text2midi/internal/style"
	"github.com/ShowerBandV/text2midi/internal/user"
)

func main() {
	llm.LoadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	outputDir := "./generated"
	fs := store.NewFileStore(outputDir)
	libDir := "./dna_library"
	local := os.Getenv("OPENAI_API_KEY") == ""

	mux := http.NewServeMux()
	srv := &Server{fs: fs, outputDir: outputDir, libDir: libDir, local: local}

	if local {
		log.Println("⚡ Local mode (no API key) — rule-based generation only")
	} else {
		log.Println("🔑 API key detected — LLM-powered generation available")
	}

	// ── User system ──────────────────────────────────────────────
	if err := user.InitDB("./data/text2midi_users.db"); err != nil {
		log.Printf("⚠️  User DB init failed (user features disabled): %v", err)
	} else {
		log.Println("👤 User system ready")

		mux.HandleFunc("POST /api/user/register", srv.handleRegister)
		mux.HandleFunc("POST /api/user/login", srv.handleLogin)
		mux.HandleFunc("POST /api/user/logout", srv.handleLogout)
		mux.HandleFunc("GET /api/user/prefs", srv.requireAuth(srv.handleGetPrefs))
		mux.HandleFunc("PUT /api/user/prefs", srv.requireAuth(srv.handleSavePrefs))
		mux.HandleFunc("GET /api/user/history", srv.requireAuth(srv.handleHistory))
		mux.HandleFunc("GET /api/user/history/{id}", srv.requireAuth(srv.handleHistoryDetail))
		mux.HandleFunc("GET /api/user/credits", srv.requireAuth(srv.handleCredits))
		mux.HandleFunc("POST /api/user/credits/add", srv.requireAuth(srv.handleAddCredits))
	}

	mux.HandleFunc("GET /api/info", srv.handleInfo)
	mux.HandleFunc("POST /api/generate", srv.handleGenerate)
	mux.HandleFunc("GET /api/files", srv.handleFileList)
	mux.HandleFunc("GET /api/files/{id}", srv.handleDownload)

	// DNA endpoints.
	mux.HandleFunc("POST /api/dna/extract", srv.handleDNAExtract)
	mux.HandleFunc("GET /api/dna/library", srv.handleDNALibraryList)
	mux.HandleFunc("GET /api/dna/library/{name}", srv.handleDNALibraryGet)

	addr := ":" + port
	log.Printf("🚀 Server starting on %s", addr)
	log.Printf("   POST /api/generate  --generate a beat")
	log.Printf("   GET  /api/info       --style constraints")
	log.Printf("   GET  /api/files/{id} --download MIDI file")

	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// ─── Auth middleware ────────────────────────────────────────────────

func (srv *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Also check cookie.
			if c, err := r.Cookie("session"); err == nil {
				token = c.Value
			}
		}
		if token == "" {
			writeJSON(w, 401, map[string]string{"error": "authentication required"})
			return
		}
		// Strip "Bearer " prefix.
		token = strings.TrimPrefix(token, "Bearer ")

		u, err := user.ValidateSession(token)
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": "invalid or expired session"})
			return
		}
		// Store user ID in request context.
		ctx := r.Context()
		ctx = contextWithUser(ctx, u)
		next(w, r.WithContext(ctx))
	}
}

type contextKey string

type userCtxKey struct{}

func contextWithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

func userFromContext(ctx context.Context) *user.User {
	u, _ := ctx.Value(userCtxKey{}).(*user.User)
	return u
}

// ─── Auth handlers ─────────────────────────────────────────────────

func (srv *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	u, err := user.Register(req.Username, req.Password)
	if err != nil {
		writeJSON(w, 409, map[string]string{"error": err.Error()})
		return
	}
	// Auto-login: create session.
	_, token, _ := user.Login(req.Username, req.Password)
	writeJSON(w, 201, map[string]any{
		"user":    u,
		"token":   token,
		"credits": user.CheckCredits(u.ID),
	})
}

func (srv *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	u, token, err := user.Login(req.Username, req.Password)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{
		"user":    u,
		"token":   token,
		"credits": user.CheckCredits(u.ID),
	})
}

func (srv *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token != "" {
		user.Logout(token)
	}
	writeJSON(w, 200, map[string]string{"status": "logged out"})
}

func (srv *Server) handleGetPrefs(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	prefs := user.GetPrefs(u.ID)
	writeJSON(w, 200, prefs)
}

func (srv *Server) handleSavePrefs(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var prefs user.Prefs
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := user.SavePrefs(u.ID, &prefs); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "saved"})
}

func (srv *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	entries, err := user.GetHistory(u.ID, 20)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"history": entries})
}

func (srv *Server) handleHistoryDetail(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	entry, err := user.GetHistoryEntry(id)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, 200, entry)
}

func (srv *Server) handleCredits(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	credits := user.CheckCredits(u.ID)
	writeJSON(w, 200, map[string]any{"credits": credits})
}

func (srv *Server) handleAddCredits(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		writeJSON(w, 400, map[string]string{"error": "invalid amount"})
		return
	}
	user.AddCredits(u.ID, req.Amount)
	writeJSON(w, 200, map[string]any{"credits": user.CheckCredits(u.ID)})
}

// Server holds shared state for HTTP handlers.
type Server struct {
	fs        *store.FileStore
	outputDir string
	libDir    string
	local     bool
	mu        sync.Mutex // protects concurrent generation
}

// --- Types ---

type InfoResponse struct {
	Styles []string          `json:"styles"`  // all available style keys
	Tiers  map[string]int    `json:"tiers"`   // tier -> maxBars
}

type GenerateRequest struct {
	Prompt          string             `json:"prompt"`
	Style           string             `json:"style"`
	BPM             int                `json:"bpm"`
	Bars            int                `json:"bars"`
	Key             string             `json:"key"`
	Tier            string             `json:"tier,omitempty"`          // default "free"
	FeatureOverride *schema.FeatureVector `json:"feature_override,omitempty"` // override feature_vector dimensions
	Seed            *int64             `json:"seed,omitempty"`          // random seed for reproducible variation
}

type GenerateResponse struct {
	Success  bool               `json:"success"`
	FileID   string             `json:"fileId,omitempty"`
	FileName string             `json:"fileName,omitempty"`
	FileSize int64              `json:"fileSize,omitempty"`
	Duration float64            `json:"durationSeconds"`
	Tracks   int                `json:"tracks"`
	Credits  int                `json:"credits,omitempty"`
	Meta     *midi.RenderResult `json:"meta,omitempty"`
	Error    string             `json:"error,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	styles := style.All()
	keys := make([]string, 0, len(styles))
	for k := range styles {
		keys = append(keys, k)
	}
	resp := InfoResponse{
		Styles: keys,
		Tiers: map[string]int{
			"free":  8,
			"basic": 16,
			"pro":   32,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	// Require authentication. Unauthenticated users cannot generate.
	u := userFromContext(r.Context())
	if u == nil {
		writeJSON(w, 401, map[string]any{"error": "login required — register/login first"})
		return
	}
	credits := user.CheckCredits(u.ID)
	if credits <= 0 {
		writeJSON(w, 402, map[string]any{
			"error": "out of generations — add credits via POST /api/user/credits/add",
		})
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	// Defaults.
	if req.Tier == "" {
		req.Tier = "free"
	}
	if req.Key == "" {
		req.Key = "C minor"
	}

	// Validate bars against tier limits.
	maxBars := 16
	switch req.Tier {
	case "free":
		maxBars = 8
	case "basic":
		maxBars = 16
	case "pro":
		maxBars = 32
	}
	if req.Bars > maxBars {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("tier %q allows max %d bars, got %d", req.Tier, maxBars, req.Bars),
		})
		return
	}

	// Generate (serialized to prevent resource contention).
	s.mu.Lock()
	result, err := s.generate(req)
	s.mu.Unlock()

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// Consume credit after successful generation.
	if remaining, ok := user.ConsumeCredit(u.ID); ok {
			writeJSON(w, http.StatusOK, GenerateResponse{
				FileID:   result.FileID,
				FileName: result.FileName,
				FileSize: result.FileSize,
				Duration: result.Duration,
				Tracks:   result.Tracks,
				Meta:     result.Meta,
				Credits:  remaining,
			})
			return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	records, err := s.fs.ListFiles()
	if err != nil {
		writeJSON(w, http.StatusOK, []store.FileRecord{})
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing id"})
		return
	}

	data, record, err := s.fs.LoadFile(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "file not found"})
		return
	}

	w.Header().Set("Content-Type", "audio/midi")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.FileName))
	w.Write(data)
}

// --- Core generation ---

// ─── DNA Handlers ─────────────────────────────────────────────────

type DNAExtractRequest struct {
	EventsByTrack map[string][]schema.NoteEvent `json:"events_by_track"`
	TotalBars     int                            `json:"total_bars"`
	Key           string                         `json:"key"`
}

func (s *Server) handleDNAExtract(w http.ResponseWriter, r *http.Request) {
	var req DNAExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}
	ext := musicdna.NewExtractor()
	dna := ext.Extract(req.EventsByTrack, req.TotalBars, req.Key)
	quality := musicdna.ScoreTemplate(dna)
	writeJSON(w, http.StatusOK, map[string]any{
		"dna":     dna,
		"quality": quality,
	})
}

func (s *Server) handleDNALibraryList(w http.ResponseWriter, r *http.Request) {
	lib := musicdna.NewLibrary(s.libDir)
	templates, err := lib.List("")
	if err != nil {
		writeJSON(w, http.StatusOK, []musicdna.DNATemplate{})
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleDNALibraryGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing name"})
		return
	}
	lib := musicdna.NewLibrary(s.libDir)
	tmpl, err := lib.Load(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "template not found"})
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// generateLocal uses rule-based generation without any LLM calls.
func (s *Server) generateLocal(req GenerateRequest) (*GenerateResponse, error) {
	ctx := composer.NewDefaultContext(req.Bars, req.BPM).
		WithStyle(0.3, 0.6, 0.4, 0.3)
	ctx.Motif = []int{0, 2, 4, 3, 0}
	events := composer.ComposeSongWithContext(ctx)

	// Build a minimal MidiIR from the generated events.
	beatsPerBar := 4
	totalBeats := req.Bars * beatsPerBar
	var tracks []schema.TrackIR
	trackID := 0

	// Default arrangement for local mode.
	arrangement := []struct {
		id, name, role string
		channel, prog, vol, pan int
	}{
		{"drums", "Drums", "drums", 9, 0, 100, 64},
		{"bass", "Bass", "bass", 1, 34, 90, 64},
		{"lead", "Lead", "lead", 4, 89, 85, 64},
		{"pad", "Pad", "Pad", 5, 91, 70, 64},
		{"fx", "FX", "fx", 6, 96, 60, 64},
	}

	for _, at := range arrangement {
		ev, ok := events[at.id]
		if !ok || len(ev) == 0 {
			continue
		}
		prog := at.prog
		tracks = append(tracks, schema.TrackIR{
			ID: at.id, Name: at.name, Role: at.role,
			Channel: at.channel, Program: &prog,
			Volume: at.vol, Pan: at.pan, Enabled: true,
			IsCoreTrack: at.role == "drums" || at.role == "bass",
			Events: ev,
		})
		trackID++
	}

	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			Title:        req.Prompt,
			BPM:          req.BPM,
			TicksPerBeat: 480,
			TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
			KeySignature:  req.Key,
			TotalBars:     req.Bars,
			BeatsPerBar:   beatsPerBar,
			TotalBeats:    totalBeats,
		},
		Tracks: tracks,
	}

	outputPath := filepath.Join(s.outputDir, "local_gen.mid")
	renderResult, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	midiData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}
	fileID := fmt.Sprintf("local_%s", randID(4))
	record, err := s.fs.SaveFile(fileID, midiData, renderResult)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	// Extract and save DNA.
	saveDir := filepath.Join(s.outputDir, record.ID)
	musicdna.SaveDNAIfValid(events, req.Bars, req.Key, saveDir)

	return &GenerateResponse{
		Success:  true,
		FileID:   record.ID,
		FileName: record.FileName,
		FileSize: record.FileSize,
		Duration: renderResult.DurationSeconds,
		Tracks:   renderResult.TotalTracks,
		Meta:     renderResult,
	}, nil
}

func (s *Server) generate(req GenerateRequest) (*GenerateResponse, error) {
	// Local mode: skip LLM, use rule-based generation.
	if s.local {
		return s.generateLocal(req)
	}

	client, err := llm.NewClient()
	if err != nil {
		return nil, fmt.Errorf("LLM client: %w", err)
	}

	// 1. ParseIntent.
	intentResult, err := agent.ParseIntent(client, req.Prompt, false, nil)
	if err != nil {
		return nil, fmt.Errorf("intent: %w", err)
	}

	// 1b. Apply feature_override if provided.
	if req.FeatureOverride != nil {
		intentMap, ok := intentResult["intent"].(map[string]any)
		if ok {
			fv, hasFV := intentMap["feature_vector"].(map[string]any)
			if !hasFV {
				fv = make(map[string]any)
			}
			if req.FeatureOverride.Darkness != 0 || req.FeatureOverride.Energy != 0 ||
				req.FeatureOverride.Acousticness != 0 || req.FeatureOverride.Density != 0 ||
				req.FeatureOverride.RhythmicComplexity != 0 || req.FeatureOverride.Tension != 0 ||
				req.FeatureOverride.LoFi != 0 {
				fv["darkness"] = req.FeatureOverride.Darkness
				fv["energy"] = req.FeatureOverride.Energy
				fv["acousticness"] = req.FeatureOverride.Acousticness
				fv["density"] = req.FeatureOverride.Density
				fv["rhythmic_complexity"] = req.FeatureOverride.RhythmicComplexity
				fv["tension"] = req.FeatureOverride.Tension
				fv["lo_fi"] = req.FeatureOverride.LoFi
				intentMap["feature_vector"] = fv
				log.Printf("  feature_override applied: %+v", req.FeatureOverride)
			}
		}
	}

	// 2. PlanSong.
	plan, songPlanRaw, err := agent.PlanSong(client, intentResult)
	if err != nil {
		return nil, fmt.Errorf("song plan: %w", err)
	}

	// 2b. Set feature_vector on SongPlan from parsed intent.
	plan.FeatureVector = agent.ParseFeatureVectorFromIntent(intentResult)

	// 3. PlanArrangement.
	arr, _, err := agent.PlanArrangement(client, intentResult, songPlanRaw, true)
	if err != nil {
		return nil, fmt.Errorf("arrangement: %w", err)
	}

	// 4. Generate patterns (beat template).
	sd := fmt.Sprintf("%s beat, BPM %d", req.Style, req.BPM)
	eventsByTrack, ccEvents, pbEvents, err := agent.GeneratePatterns(client, req.Prompt, req.Style, sd,
		plan.Key.Root+" "+plan.Key.Mode, plan.BPM, plan.TotalBars, plan.FeatureVector)
	if err != nil {
		return nil, fmt.Errorf("patterns: %w", err)
	}

	// 4b. Generate chord pad + auto-add missing tracks.
	agent.GenerateChordPad(plan, eventsByTrack)
	existing := map[string]bool{}
	for _, t := range arr.Tracks {
		existing[t.ID] = true
	}
	type autoTrack struct {
		id, name, role string
		channel, prog, vol, pan int
	}
	for _, c := range []autoTrack{
		{"bass", "Bass", "bass", 1, 34, 90, 64},
		{"drums", "Drums", "drums", 9, 0, 100, 64},
		{"chords", "Chords", "harmony", 4, 89, 75, 64},
		{"pad", "Pad", "Pad", 5, 91, 70, 64},
	} {
		if existing[c.id] { continue }
		ev := eventsByTrack[c.id]
		if ev == nil || len(ev) == 0 { continue }
		prog := c.prog
		arr.Tracks = append(arr.Tracks, schema.ArrangementTrack{
			ID: c.id, Name: c.name, Role: c.role, Enabled: true,
			IsCoreTrack: false, GenerationStrategy: "auto",
			Channel: c.channel, Program: &prog, Volume: c.vol, Pan: c.pan,
		})
		log.Printf("  auto-added track %s (%d events)", c.id, len(ev))
	}

	// 5. Assemble MidiIR.
	beatsPerBar := 4
	totalBeats := plan.TotalBars * beatsPerBar
	var tracks []schema.TrackIR
	for _, at := range arr.Tracks {
		if !at.Enabled {
			continue
		}
		events := lookupEvents(eventsByTrack, at.ID, at.Role)
		tracks = append(tracks, schema.TrackIR{
			ID: at.ID, Name: at.Name, Role: at.Role,
			Channel: at.Channel, Program: at.Program,
			Volume: at.Volume, Pan: at.Pan, Enabled: true,
			IsCoreTrack: at.IsCoreTrack, Events: events,
			CCEvents: ccEvents,
			PitchBendEvents: pbEvents,
		})
	}

	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			Title:        plan.Title,
			BPM:          plan.BPM,
			TicksPerBeat: 480,
			TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
			KeySignature:  fmt.Sprintf("%s %s", plan.Key.Root, plan.Key.Mode),
			TotalBars:     plan.TotalBars,
			BeatsPerBar:   beatsPerBar,
			TotalBeats:    totalBeats,
			Loopable:      plan.Loopable,
		},
		Tracks: tracks,
	}

	// 6. Render.
	outputPath := filepath.Join(s.outputDir, plan.Title+".mid")
	renderResult, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	// 7. Save to file store.
	midiData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}
	fileID := fmt.Sprintf("%s_%s", plan.Title, randID(8))
	record, err := s.fs.SaveFile(fileID, midiData, renderResult)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	// Extract and save MusicDNA.
	saveDir := filepath.Join(s.outputDir, record.ID)
	musicdna.SaveDNAIfValid(eventsByTrack, plan.TotalBars,
		fmt.Sprintf("%s %s", plan.Key.Root, plan.Key.Mode), saveDir)

	return &GenerateResponse{
		Success:  true,
		FileID:   record.ID,
		FileName: record.FileName,
		FileSize: record.FileSize,
		Duration: renderResult.DurationSeconds,
		Tracks:   renderResult.TotalTracks,
		Meta:     renderResult,
	}, nil
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func randID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// withCORS wraps a handler with permissive CORS headers (for local dev).
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// lookupEvents finds events for a track by ID first, then falls back to role-based lookup.
func lookupEvents(eventsByTrack map[string][]schema.NoteEvent, id, role string) []schema.NoteEvent {
	if ev, ok := eventsByTrack[id]; ok && len(ev) > 0 {
		return ev
	}
	roleMap := map[string]string{
		"melody": "lead", "lead": "lead", "piano": "lead", "keys": "lead", "synth": "lead",
		"chords": "lead", "harmony": "lead", "pad": "lead", "strings": "lead",
		"distorted_guitar": "lead", "lead_guitar": "lead", "guitar": "lead", "rhythm_guitar": "lead",
		"bass": "bass", "bassline": "bass", "synth bass": "bass",
		"rhythm": "drums", "drums": "drums", "percussion": "drums", "beat": "drums",
	}
	if beatID, ok := roleMap[role]; ok {
		if ev, ok := eventsByTrack[beatID]; ok && len(ev) > 0 {
			return ev
		}
	}
	for _, stdID := range []string{"lead", "drums", "bass"} {
		if ev, ok := eventsByTrack[stdID]; ok && len(ev) > 0 {
			return ev
		}
	}
	return []schema.NoteEvent{}
}


