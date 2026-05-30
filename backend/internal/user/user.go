package user

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DB is the SQLite database handle.
var DB *sql.DB

// InitDB opens (or creates) the SQLite database and runs migrations.
func InitDB(dbPath string) error {
	if dbPath == "" {
		dbPath = filepath.Join(os.TempDir(), "text2midi_users.db")
	}
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for concurrent reads.
	DB.Exec("PRAGMA journal_mode=WAL")

	// Run migrations.
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return fmt.Errorf("migration: %w", err)
		}
	}
	return nil
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		username    TEXT NOT NULL UNIQUE,
		password    TEXT NOT NULL,
		salt        TEXT NOT NULL,
		creates     INTEGER NOT NULL DEFAULT 5,  -- free generations remaining
		created_at  TEXT NOT NULL DEFAULT (datetime('now')),
		last_login  TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token       TEXT PRIMARY KEY,
		user_id     INTEGER NOT NULL REFERENCES users(id),
		created_at  TEXT NOT NULL DEFAULT (datetime('now')),
		expires_at  TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS user_prefs (
		user_id     INTEGER PRIMARY KEY REFERENCES users(id),
		style       TEXT DEFAULT '',
		key_name    TEXT DEFAULT 'C minor',
		bpm         INTEGER DEFAULT 140,
		bars        INTEGER DEFAULT 32,
		flat_vel    INTEGER DEFAULT 100,
		chaos       REAL DEFAULT 0.0,
		progression TEXT DEFAULT '',
		mode        TEXT DEFAULT '',
		loopable    INTEGER DEFAULT 0,
		pentatonic  INTEGER DEFAULT 0,
		updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS generation_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL REFERENCES users(id),
		prompt      TEXT,
		style       TEXT,
		key_name    TEXT,
		bpm         INTEGER,
		bars        INTEGER,
		midi_path   TEXT,
		duration_s  REAL,
		note_count  INTEGER,
		created_at  TEXT NOT NULL DEFAULT (datetime('now'))
	)`,
}

// ─── User operations ────────────────────────────────────────────────

// User represents a registered user.
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// Register creates a new user account. Returns the user or an error.
func Register(username, password string) (*User, error) {
	if username == "" || len(username) < 3 {
		return nil, fmt.Errorf("username must be at least 3 characters")
	}
	if len(password) < 4 {
		return nil, fmt.Errorf("password must be at least 4 characters")
	}

	salt := randomHex(16)
	hash := hashPassword(password, salt)

	res, err := DB.Exec(
		"INSERT INTO users (username, password, salt) VALUES (?, ?, ?)",
		username, hash, salt,
	)
	if err != nil {
		// Check for duplicate.
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("username already taken")
		}
		return nil, fmt.Errorf("register: %w", err)
	}

	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username}, nil
}

// Login authenticates a user and returns a session token.
func Login(username, password string) (*User, string, error) {
	var id int64
	var hash, salt string
	err := DB.QueryRow(
		"SELECT id, password, salt FROM users WHERE username = ?", username,
	).Scan(&id, &hash, &salt)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("invalid username or password")
	}
	if err != nil {
		return nil, "", fmt.Errorf("login: %w", err)
	}

	if hashPassword(password, salt) != hash {
		return nil, "", fmt.Errorf("invalid username or password")
	}

	// Create session token (valid for 30 days).
	token := randomHex(32)
	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, err = DB.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, id, expires,
	)
	if err != nil {
		return nil, "", fmt.Errorf("session: %w", err)
	}

	// Update last login.
	DB.Exec("UPDATE users SET last_login = datetime('now') WHERE id = ?", id)

	return &User{ID: id, Username: username}, token, nil
}

// ValidateSession checks a token and returns the user if valid.
func ValidateSession(token string) (*User, error) {
	var id int64
	var username, expires string
	err := DB.QueryRow(
		`SELECT u.id, u.username, s.expires_at
		 FROM sessions s JOIN users u ON s.user_id = u.id
		 WHERE s.token = ?`, token,
	).Scan(&id, &username, &expires)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid session")
	}
	if err != nil {
		return nil, fmt.Errorf("session: %w", err)
	}

	exp, _ := time.Parse(time.RFC3339, expires)
	if time.Now().UTC().After(exp) {
		DB.Exec("DELETE FROM sessions WHERE token = ?", token)
		return nil, fmt.Errorf("session expired")
	}

	return &User{ID: id, Username: username}, nil
}

// Logout deletes the session token.
func Logout(token string) {
	DB.Exec("DELETE FROM sessions WHERE token = ?", token)
}

// ─── Preferences ────────────────────────────────────────────────────

// Prefs holds user generation preferences.
type Prefs struct {
	Style       string  `json:"style"`
	KeyName     string  `json:"key"`
	BPM         int     `json:"bpm"`
	Bars        int     `json:"bars"`
	FlatVel     int     `json:"flatVel"`
	Chaos       float64 `json:"chaos"`
	Progression string  `json:"progression"`
	Mode        string  `json:"mode"`
	Loopable    bool    `json:"loopable"`
	Pentatonic  bool    `json:"pentatonic"`
}

// GetPrefs returns saved preferences for a user (or defaults).
func GetPrefs(userID int64) *Prefs {
	p := &Prefs{Style: "", KeyName: "C minor", BPM: 140, Bars: 32, FlatVel: 100}
	var loop, pent int
	err := DB.QueryRow(
		"SELECT style, key_name, bpm, bars, flat_vel, chaos, progression, mode, loopable, pentatonic FROM user_prefs WHERE user_id = ?",
		userID,
	).Scan(&p.Style, &p.KeyName, &p.BPM, &p.Bars, &p.FlatVel, &p.Chaos, &p.Progression, &p.Mode, &loop, &pent)
	if err != nil {
		return p // return defaults
	}
	p.Loopable = loop != 0
	p.Pentatonic = pent != 0
	return p
}

// SavePrefs saves user preferences.
func SavePrefs(userID int64, p *Prefs) error {
	loop := 0
	if p.Loopable {
		loop = 1
	}
	pent := 0
	if p.Pentatonic {
		pent = 1
	}
	_, err := DB.Exec(
		`INSERT OR REPLACE INTO user_prefs
		 (user_id, style, key_name, bpm, bars, flat_vel, chaos, progression, mode, loopable, pentatonic, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		userID, p.Style, p.KeyName, p.BPM, p.Bars, p.FlatVel, p.Chaos, p.Progression, p.Mode, loop, pent,
	)
	return err
}

// ─── Generations / Credits ──────────────────────────────────────────

// CheckCredits returns how many free generations a user has left.
func CheckCredits(userID int64) int {
	var c int
	err := DB.QueryRow("SELECT creates FROM users WHERE id = ?", userID).Scan(&c)
	if err != nil {
		return 0
	}
	return c
}

// ConsumeCredit decrements the remaining free generations by 1.
// Returns (remaining, canProceed) where canProceed is false if already 0.
func ConsumeCredit(userID int64) (int, bool) {
	remaining := CheckCredits(userID)
	if remaining <= 0 {
		// Allow at most 1 extra overage; check if has any credits or needs purchase.
		return 0, false
	}
	DB.Exec("UPDATE users SET creates = creates - 1 WHERE id = ? AND creates > 0", userID)
	return remaining - 1, true
}

// AddCredits adds paid generations to a user's account.
func AddCredits(userID int64, amount int) {
	DB.Exec("UPDATE users SET creates = creates + ? WHERE id = ?", amount, userID)
}

// ─── Generation History ─────────────────────────────────────────────

// HistoryEntry represents one generation.
type HistoryEntry struct {
	ID         int64   `json:"id"`
	Prompt     string  `json:"prompt,omitempty"`
	Style      string  `json:"style"`
	KeyName    string  `json:"key"`
	BPM        int     `json:"bpm"`
	Bars       int     `json:"bars"`
	MIDIPath   string  `json:"midiPath,omitempty"`
	DurationS  float64 `json:"duration,omitempty"`
	NoteCount  int     `json:"noteCount,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}

// AddHistory records a generation.
func AddHistory(userID int64, prompt, style, keyName string, bpm, bars int, midiPath string, durS float64, notes int) error {
	_, err := DB.Exec(
		`INSERT INTO generation_history (user_id, prompt, style, key_name, bpm, bars, midi_path, duration_s, note_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, prompt, style, keyName, bpm, bars, midiPath, durS, notes,
	)
	return err
}

// GetHistoryEntry returns a single generation history entry.
func GetHistoryEntry(id int64) (*HistoryEntry, error) {
	var e HistoryEntry
	var prompt, midiPath sql.NullString
	var dur sql.NullFloat64
	var notes sql.NullInt64
	err := DB.QueryRow(
		`SELECT id, prompt, style, key_name, bpm, bars, midi_path, duration_s, note_count, created_at
		 FROM generation_history WHERE id = ?`, id,
	).Scan(&e.ID, &prompt, &e.Style, &e.KeyName, &e.BPM, &e.Bars, &midiPath, &dur, &notes, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not found")
	}
	if err != nil {
		return nil, err
	}
	e.Prompt = prompt.String
	e.MIDIPath = midiPath.String
	e.DurationS = dur.Float64
	e.NoteCount = int(notes.Int64)
	return &e, nil
}

// GetHistory returns recent generations for a user.
func GetHistory(userID int64, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := DB.Query(
		`SELECT id, prompt, style, key_name, bpm, bars, midi_path, duration_s, note_count, created_at
		 FROM generation_history WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var prompt, midiPath sql.NullString
		var dur sql.NullFloat64
		var notes sql.NullInt64
		if err := rows.Scan(&e.ID, &prompt, &e.Style, &e.KeyName, &e.BPM, &e.Bars, &midiPath, &dur, &notes, &e.CreatedAt); err != nil {
			continue
		}
		e.Prompt = prompt.String
		e.MIDIPath = midiPath.String
		e.DurationS = dur.Float64
		e.NoteCount = int(notes.Int64)
		entries = append(entries, e)
	}
	return entries, nil
}

// ─── Helpers ────────────────────────────────────────────────────────

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password, salt string) string {
	h := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(h[:])
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && search(s, sub) >= 0
}

func search(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ─── JSON helpers ───────────────────────────────────────────────────

// MergePrefsJSON merges saved prefs with incoming request JSON.
func MergePrefsJSON(userID int64, reqBody []byte) *Prefs {
	prefs := GetPrefs(userID)
	if len(reqBody) == 0 {
		return prefs
	}
	// Override saved prefs with any fields in the request.
	var override map[string]any
	if err := json.Unmarshal(reqBody, &override); err != nil {
		return prefs
	}
	if v, ok := override["style"]; ok {
		prefs.Style = fmt.Sprint(v)
	}
	if v, ok := override["key"]; ok {
		prefs.KeyName = fmt.Sprint(v)
	}
	if v, ok := override["bpm"]; ok {
		if f, ok := v.(float64); ok {
			prefs.BPM = int(f)
		}
	}
	if v, ok := override["bars"]; ok {
		if f, ok := v.(float64); ok {
			prefs.Bars = int(f)
		}
	}
	if v, ok := override["chaos"]; ok {
		if f, ok := v.(float64); ok {
			prefs.Chaos = f
		}
	}
	if v, ok := override["flatVel"]; ok {
		if f, ok := v.(float64); ok {
			prefs.FlatVel = int(f)
		}
	}
	return prefs
}
