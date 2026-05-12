package models

import "time"

// Mode represents the interview question mode.
type Mode string

const (
	ModeBehavioral Mode = "behavioral"
	ModeTechnical  Mode = "technical"
	ModeMixed      Mode = "mixed"
	ModeSysDesign  Mode = "sysdesign"
)

// Difficulty represents the target difficulty level.
type Difficulty string

const (
	DiffJunior Difficulty = "junior"
	DiffMid    Difficulty = "mid"
	DiffSenior Difficulty = "senior"
	DiffStaff  Difficulty = "staff"
)

// Status represents the current state of a session.
type Status string

const (
	StatusReady     Status = "ready"
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
)

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Score holds evaluation results for a single answer.
type Score struct {
	Score              int      `json:"score"`
	Strengths          []string `json:"strengths"`
	Gaps               []string `json:"gaps"`
	IdealAnswerSummary string   `json:"ideal_answer_summary"`
	FollowUpWarranted  bool     `json:"follow_up_warranted"`
	FollowUpQuestion   string   `json:"follow_up_question"`
}

// Turn represents one Q&A exchange in the interview.
type Turn struct {
	Index     int           `json:"index"`
	Question  string        `json:"question"`
	Answer    string        `json:"answer"`
	Score     *Score        `json:"score,omitempty"`
	FollowUps []FollowUp    `json:"follow_ups,omitempty"`
	Skipped   bool          `json:"skipped"`
	Duration  time.Duration `json:"duration_ms"`
	StartedAt time.Time     `json:"started_at"`
}

// FollowUp holds a follow-up Q&A pair.
type FollowUp struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Summary holds the end-of-session evaluation summary.
type Summary struct {
	OverallScore           int      `json:"overall_score"`
	OverallAssessment      string   `json:"overall_assessment"`
	TopStrengths           []string `json:"top_strengths"`
	CriticalGaps           []string `json:"critical_gaps"`
	RecommendedStudyTopics []string `json:"recommended_study_topics"`
	HiringRecommendation   string   `json:"hiring_recommendation"`
}

// Session holds the full state of an interview session.
type Session struct {
	ID           string     `json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ResumeHash   string     `json:"resume_hash"`
	ResumePath   string     `json:"resume_path"`
	Role         string     `json:"role"`
	Mode         Mode       `json:"mode"`
	Difficulty   Difficulty `json:"difficulty"`
	NumQuestions int        `json:"num_questions"`
	Questions    []string   `json:"questions"`
	Turns        []Turn     `json:"turns"`
	History      []Message  `json:"history"`
	Status       Status     `json:"status"`
	Summary      *Summary   `json:"summary,omitempty"`
	TokensUsed   int        `json:"tokens_used"`
}

// SessionMeta is a lightweight summary used for listing sessions.
type SessionMeta struct {
	ID           string     `json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	Role         string     `json:"role"`
	Mode         Mode       `json:"mode"`
	Difficulty   Difficulty `json:"difficulty"`
	Status       Status     `json:"status"`
	OverallScore *int       `json:"overall_score,omitempty"`
	NumTurns     int        `json:"num_turns"`
}

// CurrentTurnIndex returns the index of the current turn (0-based).
func (s *Session) CurrentTurnIndex() int {
	return len(s.Turns)
}

// IsComplete returns true when all questions have been answered or the session was aborted.
func (s *Session) IsComplete() bool {
	return s.Status == StatusCompleted || s.Status == StatusAborted
}

// MarkAborted sets the session status to aborted.
func (s *Session) MarkAborted() {
	s.Status = StatusAborted
	s.UpdatedAt = time.Now()
}

// MarkCompleted sets the session as completed with the given summary.
func (s *Session) MarkCompleted(summary Summary) {
	s.Status = StatusCompleted
	s.Summary = &summary
	s.UpdatedAt = time.Now()
}
