package interview

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/absolutezero000/prep/internal/models"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templateNames = []string{
	"templates/system.tmpl",
	"templates/question_gen.tmpl",
	"templates/evaluate.tmpl",
	"templates/followup.tmpl",
	"templates/hint.tmpl",
	"templates/summarize.tmpl",
}

var parsedTemplates *template.Template

func init() {
	parsedTemplates = template.Must(template.ParseFS(templateFS, templateNames...))
}

// SystemData is the template data for the system prompt.
type SystemData struct {
	Resume       string
	Role         string
	Mode         string
	Difficulty   string
	NumQuestions int
}

// QuestionGenData is the template data for question generation.
type QuestionGenData struct {
	NumQuestions int
	Difficulty   string
	Role         string
	Resume       string
}

// EvaluateData is the template data for answer evaluation.
type EvaluateData struct {
	Question      string
	Answer        string
	Role          string
	Context       string // full conversation context for follow-up evaluations
	FollowUpDepth int    // which follow-up this is (0 = initial answer)
}

// FollowUpData is the template data for follow-up question generation.
type FollowUpData struct {
	Question      string
	Answer        string
	PreviousScore int
}

// SummarizeData is the template data for session summarization.
type SummarizeData struct {
	Turns []models.Turn
}

// RenderTemplate renders a named template with the given data.
func RenderTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := parsedTemplates.ExecuteTemplate(&buf, name+".tmpl", data); err != nil {
		return "", fmt.Errorf("rendering template %s: %w", name, err)
	}
	return buf.String(), nil
}
