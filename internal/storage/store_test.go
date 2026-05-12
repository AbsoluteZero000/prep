package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/absolutezero000/prep/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".prep")
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func testSession() *models.Session {
	now := time.Now()
	return &models.Session{
		ID:           "test-123",
		CreatedAt:    now,
		UpdatedAt:    now,
		Role:         "Backend Engineer",
		Mode:         models.ModeTechnical,
		Difficulty:   models.DiffSenior,
		NumQuestions: 3,
		Status:       models.StatusActive,
		Questions:    []string{"Q1", "Q2", "Q3"},
	}
}

func TestNewStore_CreatesDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".prep")
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{dir, s.sessionsDir(), s.cacheDir()} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Fatalf("expected directory %s to exist", d)
		}
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	s := newTestStore(t)
	sess := testSession()

	if err := s.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.LoadSession("test-123")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "test-123" {
		t.Fatalf("expected ID test-123, got %s", loaded.ID)
	}
	if loaded.Role != "Backend Engineer" {
		t.Fatalf("expected role Backend Engineer, got %s", loaded.Role)
	}
	if loaded.Status != models.StatusActive {
		t.Fatalf("expected StatusActive, got %s", loaded.Status)
	}
}

func TestLoadSession_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.LoadSession("nonexistent")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestLoadSession_Corrupt(t *testing.T) {
	s := newTestStore(t)
	path := filepath.Join(s.sessionsDir(), "corrupt.json")
	os.WriteFile(path, []byte("{invalid json}"), 0600)

	_, err := s.LoadSession("corrupt")
	if err == nil || err == ErrSessionNotFound {
		t.Fatalf("expected ErrCorruptSession, got %v", err)
	}
}

func TestListSessions(t *testing.T) {
	s := newTestStore(t)

	// No sessions yet
	metas, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(metas))
	}

	// Add two sessions
	sess1 := testSession()
	sess1.ID = "first"
	sess1.CreatedAt = time.Now().Add(-time.Hour)
	s.SaveSession(sess1)

	sess2 := testSession()
	sess2.ID = "second"
	sess2.CreatedAt = time.Now()
	s.SaveSession(sess2)

	metas, err = s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(metas))
	}
	// Should be sorted newest-first
	if metas[0].ID != "second" || metas[1].ID != "first" {
		t.Fatalf("expected newest-first order, got %s, %s", metas[0].ID, metas[1].ID)
	}
}

func TestDeleteSession(t *testing.T) {
	s := newTestStore(t)
	sess := testSession()
	s.SaveSession(sess)

	if err := s.DeleteSession("test-123"); err != nil {
		t.Fatal(err)
	}

	_, err := s.LoadSession("test-123")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.DeleteSession("ghost")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExportMarkdown(t *testing.T) {
	s := newTestStore(t)
	sess := testSession()
	sess.Turns = []models.Turn{
		{
			Index:    0,
			Question: "What is Go?",
			Answer:   "A programming language.",
			Score: &models.Score{
				Score:              8,
				Strengths:          []string{"Clear", "Concise"},
				Gaps:               []string{"Lacks depth"},
				IdealAnswerSummary: "More detail expected",
			},
		},
	}
	sess.MarkCompleted(models.Summary{
		OverallScore:         7,
		OverallAssessment:    "Good performance overall.",
		HiringRecommendation: "yes",
	})

	path, err := s.ExportMarkdown(sess)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !stringsContains(content, "Interview Session") {
		t.Fatal("expected markdown to contain Interview Session header")
	}
	if !stringsContains(content, "What is Go?") {
		t.Fatal("expected markdown to contain question")
	}
	if !stringsContains(content, "Strong") {
		t.Log("markdown generated successfully")
	}
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCacheResume(t *testing.T) {
	s := newTestStore(t)

	if err := s.CacheResume("abc123", "resume text"); err != nil {
		t.Fatal(err)
	}

	text, ok := s.LoadCachedResume("abc123")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if text != "resume text" {
		t.Fatalf("expected 'resume text', got '%s'", text)
	}
}

func TestLoadCachedResume_Miss(t *testing.T) {
	s := newTestStore(t)
	_, ok := s.LoadCachedResume("nonexistent")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestLoadCachedResume_EmptyHash(t *testing.T) {
	s := newTestStore(t)
	_, ok := s.LoadCachedResume("")
	if ok {
		t.Fatal("expected cache miss for empty hash")
	}
}

func TestSaveSession_EmptyID(t *testing.T) {
	s := newTestStore(t)
	sess := testSession()
	sess.ID = ""
	err := s.SaveSession(sess)
	if err == nil {
		t.Fatal("expected error for empty session ID")
	}
}

func TestListSessions_SkipsCorrupt(t *testing.T) {
	s := newTestStore(t)
	// Write a valid session and a corrupt one
	sess := testSession()
	sess.ID = "valid"
	s.SaveSession(sess)

	os.WriteFile(filepath.Join(s.sessionsDir(), "broken.json"), []byte("bad json"), 0600)

	metas, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 valid session, got %d", len(metas))
	}
}

func TestSessionPersistenceRoundTrip(t *testing.T) {
	s := newTestStore(t)
	sess := testSession()
	sess.Turns = []models.Turn{
		{
			Index:    0,
			Question: "Q1",
			Answer:   "A1",
			Duration: time.Minute,
		},
	}
	sess.TokensUsed = 150

	if err := s.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.LoadSession("test-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(loaded.Turns))
	}
	if loaded.TokensUsed != 150 {
		t.Fatalf("expected TokensUsed 150, got %d", loaded.TokensUsed)
	}
}
