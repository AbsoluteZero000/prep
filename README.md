# prep — AI Interview Coach

**prep** runs live, voice-ready mock interviews against your actual resume using OpenRouter AI. It generates tailored questions, streams real-time feedback, scores your answers, and asks follow-ups — all from the terminal.

## Features

- **Resume-aware questions** — parses PDF, DOCX, and TXT; questions are built around your actual experience
- **Streaming Q&A** — type answers in a Bubble Tea editor, get instant LLM feedback as it arrives
- **Scoring with rubric** — each answer gets 1–10 with specific strengths and gaps
- **Follow-up drill-down** — up to 3 follow-up questions per turn when the AI detects room to dig deeper
- **4 modes** — behavioral, technical, mixed, sysdesign
- **4 difficulty levels** — junior, mid, senior, staff
- **Session history** — review past interviews, export to markdown
- **Partial ID matching** — `prep review abc` matches any session starting with `abc`
- **Hint/skip/quit** — `/hint`, `/skip`, `/quit` during any turn
- **Color-coded UI** — pros green, cons red, AI feedback magenta, questions cyan

## Install

```bash
go install github.com/absolutezero000/prep@latest
```

Or build from source:

```bash
git clone https://github.com/absolutezero000/prep
cd prep
make build
./bin/prep
```

## Quick Start

```bash
# Interactive setup (prompts for API key, model, defaults)
prep config setup

# Start an interview
prep start --resume ~/resume.pdf

# With options
prep start --resume ~/resume.pdf --mode technical --difficulty senior --questions 10
```

## Commands

### `prep start [flags]`

Start a new interview session.

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --resume` | — | Path to resume (PDF, DOCX, TXT) |
| `--role` | inferred | Target job role |
| `--mode` | config default | behavioral, technical, mixed, sysdesign |
| `--difficulty` | config default | junior, mid, senior, staff |
| `-q, --questions` | 5 | Number of questions (max 15) |
| `--model` | config default | Override AI model |
| `--stream` | true | Enable streaming output |

**During an interview:**

| Command | Action |
|---------|--------|
| `hint` | Get a nudge on the current question |
| `skip` | Move to the next question |
| `quit` / `exit` | Save and exit |
| Enter twice | Submit your answer |

### `prep history [flags]`

List past sessions.

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 10 | Number of sessions |
| `--status` | — | Filter: active, completed, aborted |

### `prep review <session-id> [flags]`

Review a past session (supports partial ID matching).

| Flag | Default | Description |
|------|---------|-------------|
| `--export` | false | Export session to markdown |

### `prep config`

| Subcommand | Description |
|------------|-------------|
| `setup` | Interactive setup wizard |
| `show` | Display current config (key redacted) |
| `set-key <key>` | Set OpenRouter API key |
| `set-model <model>` | Set AI model (validated against OpenRouter) |
| `reset` | Delete configuration |

## Configuration

**Config file:** `~/.prep/config.yaml`

**Environment variables:**

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | Overrides config file key |
| `PREP_CONFIG` | Custom config file path |
| `PREP_CONFIG_DIR` | Custom data directory |
| `NO_COLOR` | Disable colored output |

## Build

```bash
make build    # build to bin/prep
make test     # run tests with race detector
make lint     # go vet + golangci-lint
make install  # build + copy to $GOPATH/bin
make clean    # remove bin/
```

## Architecture

```
cmd/            # cobra commands (start, history, review, config)
internal/
  config/       # YAML config, env override, validation
  interview/    # session engine, prompt templates, evaluation
  models/       # shared types (Session, Turn, Score, etc.)
  openrouter/   # API client adapter for revrost/go-openrouter
  resume/       # PDF/DOCX/TXT parser with heuristic validation
  storage/      # atomic file-based session CRUD + markdown export
  tokens/       # token estimation, history trimming, model limits
  ui/           # Bubble Tea textarea, spinner, colors, TTY detection
```

## Color Reference

| Element | Color |
|---------|-------|
| Questions | Cyan |
| Your input | Default (terminal white) |
| AI feedback | Magenta |
| Strengths | Green |
| Gaps | Red |
| Warnings | Yellow |
| Score (1–4) | Red |
| Score (5–7) | Yellow |
| Score (8–10) | Green |
