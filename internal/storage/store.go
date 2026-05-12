// Package storage provides persistent storage for sessions and cached data.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/absolutezero000/prep/internal/models"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrCorruptSession  = errors.New("corrupt session file")
)

// Store manages on-disk storage for prep data.
type Store struct {
	baseDir string
}

// NewStore creates a Store and ensures the required directory structure exists.
func NewStore(baseDir string) (*Store, error) {
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "sessions"),
		filepath.Join(baseDir, "resume_cache"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return nil, fmt.Errorf("creating store directory %s: %w", d, err)
		}
	}
	return &Store{baseDir: baseDir}, nil
}

func (s *Store) sessionsDir() string {
	return filepath.Join(s.baseDir, "sessions")
}

func (s *Store) cacheDir() string {
	return filepath.Join(s.baseDir, "resume_cache")
}

// SaveSession persists a session to disk as JSON.
func (s *Store) SaveSession(sess *models.Session) error {
	if sess.ID == "" {
		return errors.New("session ID is required")
	}
	sess.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	path := filepath.Join(s.sessionsDir(), sess.ID+".json")
	if err := WriteFileAtomic(path, data, 0600); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}
	return nil
}

// LoadSession reads a session from disk by ID.
func (s *Store) LoadSession(id string) (*models.Session, error) {
	path := filepath.Join(s.sessionsDir(), id+".json")
	data, err := os.ReadFile(path) //nolint:gosec // session path is within data dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}

	var sess models.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCorruptSession, path, err)
	}
	return &sess, nil
}

// ListSessions returns metadata for all sessions, sorted newest-first.
func (s *Store) ListSessions() ([]models.SessionMeta, error) {
	entries, err := os.ReadDir(s.sessionsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var metas []models.SessionMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		sess, err := s.LoadSession(id)
		if err != nil {
			continue // skip corrupt sessions silently in listing
		}

		meta := models.SessionMeta{
			ID:         sess.ID,
			CreatedAt:  sess.CreatedAt,
			Role:       sess.Role,
			Mode:       sess.Mode,
			Difficulty: sess.Difficulty,
			Status:     sess.Status,
			NumTurns:   len(sess.Turns),
		}
		if sess.Summary != nil {
			meta.OverallScore = &sess.Summary.OverallScore
		}
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt.After(metas[j].CreatedAt)
	})

	return metas, nil
}

// DeleteSession removes a session file from disk.
func (s *Store) DeleteSession(id string) error {
	path := filepath.Join(s.sessionsDir(), id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// ExportMarkdown generates a markdown summary of a session and writes it to disk.
// Returns the path to the exported file.
func (s *Store) ExportMarkdown(sess *models.Session) (string, error) {
	var b strings.Builder

	scoreStr := "—"
	if sess.Summary != nil {
		scoreStr = fmt.Sprintf("%d/10", sess.Summary.OverallScore)
	}

	fmt.Fprintf(&b, "# Interview Session — %s (%s)\n", sess.Role, sess.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(&b, "**Mode:** %s | **Difficulty:** %s | **Score:** %s\n\n", sess.Mode, sess.Difficulty, scoreStr)

	b.WriteString("## Questions & Answers\n\n")

	for i, turn := range sess.Turns {
		fmt.Fprintf(&b, "### Q%d: %s\n\n", i+1, turn.Question)
		fmt.Fprintf(&b, "**Your Answer:** %s\n\n", turn.Answer)

		if turn.Score != nil {
			fmt.Fprintf(&b, "**Score:** %d/10\n\n", turn.Score.Score)
			fmt.Fprintf(&b, "**Strengths:** %s\n\n", strings.Join(turn.Score.Strengths, ", "))
			fmt.Fprintf(&b, "**Gaps:** %s\n\n", strings.Join(turn.Score.Gaps, ", "))
		}

		if turn.Skipped {
			b.WriteString("*(skipped)*\n\n")
		}

		b.WriteString("---\n\n")
	}

	if sess.Summary != nil {
		b.WriteString("## Summary\n\n")
		b.WriteString(sess.Summary.OverallAssessment + "\n\n")
		fmt.Fprintf(&b, "**Recommendation:** %s\n", sess.Summary.HiringRecommendation)
	}

	path := filepath.Join(s.sessionsDir(), sess.ID+".md")
	if err := WriteFileAtomic(path, []byte(b.String()), 0644); err != nil {
		return "", fmt.Errorf("writing markdown export: %w", err)
	}
	return path, nil
}

// CacheResume stores parsed resume text indexed by its hash.
func (s *Store) CacheResume(hash, text string) error {
	if hash == "" {
		return nil
	}
	path := filepath.Join(s.cacheDir(), hash+".txt")
	if err := WriteFileAtomic(path, []byte(text), 0644); err != nil {
		return fmt.Errorf("caching resume: %w", err)
	}
	return nil
}

// LoadCachedResume retrieves a cached resume by hash. Returns false on miss.
func (s *Store) LoadCachedResume(hash string) (string, bool) {
	if hash == "" {
		return "", false
	}
	path := filepath.Join(s.cacheDir(), hash+".txt")
	data, err := os.ReadFile(path) //nolint:gosec // hash-based path within data dir
	if err != nil {
		return "", false
	}
	return string(data), true
}
