// Package store provides file-system storage for generated MIDI files.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yourname/text2midi/internal/midi"
)

// FileRecord stores metadata about a generated MIDI file.
type FileRecord struct {
	ID          string          `json:"id"`
	FileName    string          `json:"file_name"`
	FilePath    string          `json:"file_path"`
	FileSize    int64           `json:"file_size"`
	CreatedAt   time.Time       `json:"created_at"`
	RenderMeta  *midi.RenderResult `json:"render_meta,omitempty"`
}

// FileStore manages reading and writing MIDI files and their metadata to disk.
type FileStore struct {
	BaseDir string // root directory for all stored files
}

// NewFileStore creates a FileStore rooted at baseDir.
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{BaseDir: baseDir}
}

// SaveFile writes a MIDI binary to disk and returns a FileRecord.
// The file is saved at {BaseDir}/{id}/{id}.mid.
func (fs *FileStore) SaveFile(id string, data []byte, renderMeta *midi.RenderResult) (*FileRecord, error) {
	dir := filepath.Join(fs.BaseDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	fileName := fmt.Sprintf("%s.mid", id)
	filePath := filepath.Join(dir, fileName)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	record := &FileRecord{
		ID:         id,
		FileName:   fileName,
		FilePath:   filePath,
		FileSize:   info.Size(),
		CreatedAt:  info.ModTime(),
		RenderMeta: renderMeta,
	}

	// Write metadata JSON alongside the file.
	if err := fs.writeMeta(dir, record); err != nil {
		return nil, fmt.Errorf("write meta: %w", err)
	}

	return record, nil
}

// LoadFile reads a MIDI file from disk by id.
func (fs *FileStore) LoadFile(id string) ([]byte, *FileRecord, error) {
	record, err := fs.LoadMeta(id)
	if err != nil {
		return nil, nil, err
	}

	data, err := os.ReadFile(record.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	return data, record, nil
}

// LoadMeta reads only the metadata for a stored file.
func (fs *FileStore) LoadMeta(id string) (*FileRecord, error) {
	dir := filepath.Join(fs.BaseDir, id)
	metaPath := filepath.Join(dir, "meta.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read meta for %q: %w", id, err)
	}

	var record FileRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse meta for %q: %w", id, err)
	}

	return &record, nil
}

// ListFiles returns all stored file records.
func (fs *FileStore) ListFiles() ([]FileRecord, error) {
	entries, err := os.ReadDir(fs.BaseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list dir: %w", err)
	}

	var records []FileRecord
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(fs.BaseDir, entry.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue // skip dirs without valid meta
		}
		var rec FileRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// DeleteFile removes a stored file and its metadata.
func (fs *FileStore) DeleteFile(id string) error {
	dir := filepath.Join(fs.BaseDir, id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete %q: %w", id, err)
	}
	return nil
}

// writeMeta writes the FileRecord JSON to {dir}/meta.json.
func (fs *FileStore) writeMeta(dir string, record *FileRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	path := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	return nil
}
