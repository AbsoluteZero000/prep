# prep — AI Interview Preparation CLI

A Go CLI tool that uses OpenRouter AI to conduct personalized mock interviews based on your resume.

## Features

- Resume parser (PDF, DOCX, TXT)
- AI-generated interview questions tailored to your background
- Streaming Q&A with real-time evaluation
- Session history and markdown export
- Configurable difficulty, mode, and models

## Quick Start

```bash
# Set your API key
prep config setup

# Start an interview
prep start --resume ~/resume.pdf

# View past sessions
prep history

# Review a session
prep review <session-id>
```

## Commands

| Command | Description |
|---------|-------------|
| `prep start` | Start a new interview session |
| `prep history` | List past sessions |
| `prep review` | Review a session |
| `prep config` | Manage configuration |

## Configuration

Config file: `~/.prep/config.yaml`

Environment variables:
- `OPENROUTER_API_KEY` — overrides config file
- `PREP_CONFIG` — custom config file path
- `PREP_CONFIG_DIR` — custom data directory
- `NO_COLOR` — disables colored output

## Build

```bash
make build
make test
```
