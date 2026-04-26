# News Digest

Go module: `git.romanzipp.net/romanzipp/news`

Self-hosted, multi-user personal news digest web app in Go.

## Tech Stack

- **Backend**: Go 1.23, net/http (Go 1.22+ ServeMux)
- **Frontend**: Tailwind CSS v4, htmx (via npm), Go html/template
- **Database**: SQLite (default) or PostgreSQL
- **AI**: OpenAI-compatible (go-openai) or Anthropic (anthropic-sdk-go), switchable via AI_PROVIDER env var
- **Auth**: alexedwards/scs (sessions) + bcrypt (passwords)

## Project Structure

```
cmd/news/main.go              — entry point
internal/
  config/config.go            — env var loading
  database/                   — Open(), Migrate(), migrations SQL
  models/models.go            — domain structs
  auth/                       — password hashing, scs session middleware
  source/                     — abstract Provider interface + rss/freshrss implementations
  digest/                     — AI generation orchestration, prompt building, JSON parsing
  ai/client.go                — OpenAI-compatible wrapper
  imageproxy/proxy.go         — remote image proxy
  handlers/                   — HTTP handlers
  templates/render.go         — template loading + helpers
templates/                    — Go html/template files + partials/
static/css/                   — Tailwind input.css + output.css
static/js/                    — htmx (copied from node_modules at build)
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| alexedwards/scs/v2 | Session management |
| golang.org/x/crypto/bcrypt | Password hashing |
| sashabaranov/go-openai | OpenAI API client |
| anthropics/anthropic-sdk-go | Anthropic API client |
| mmcdole/gofeed | RSS/Atom parsing |
| mattn/go-sqlite3 | SQLite driver |
| lib/pq | PostgreSQL driver |
| robfig/cron/v3 | Scheduled tasks |
| joho/godotenv | .env loading |

## Database

Sources table is abstract — `type` field + `config` JSON blob. Adding new source types requires only a new Go provider, no schema changes.

Tables: users, sources, articles, digests, digest_items, custom_sections, section_items, interests, votes. Sessions managed by scs.

## Routes

```
Auth:     GET/POST /login, /register, POST /logout
Digest:   GET / (redirect), GET /digest/{date}, GET /digest/{date}/{id}, POST /digest/generate
Feeds:    GET/POST /feeds, POST /feeds/{id}/delete, POST /feeds/{id}/toggle
FreshRSS: GET/POST /freshrss, POST /freshrss/test
Wizard:   GET/POST /wizard
Interests: GET/POST /interests
Sections: GET/POST /sections, POST /sections/{id}/delete
Votes:    POST /votes
Proxy:    GET /proxy/image?url=...
Static:   GET /static/...
```

## AI Digest Flow

1. Fetch articles from all enabled sources (cron 05:00)
2. Gather user interests, vote history, custom sections
3. Build system prompt + article prompt
4. Batch if exceeding token limit, call AI
5. Parse structured JSON response
6. Store digest + items in single transaction

## Design

Editorial/newspaper aesthetic from design-template/project/v1-editorial.jsx:
- Colors: cream #f7f3ec, ink #1a1612, muted #6b5f52, rule #d8ccb8
- Fonts: Playfair Display (masthead/headlines), Georgia/Source Serif 4 (body)
- Layout: masthead → info strip → lead story → secondary stories → brief grid → trending → colophon
- Dark mode via prefers-color-scheme

## Development

```bash
make dev        # run with CSS watch
make build      # build binary + CSS
make css        # build CSS once
make css-watch  # watch CSS changes
```

## Code Style

- Reuse template components (partials) — never duplicate markup
- Comments only where strictly necessary
- Keep features abstract/extensible (e.g. source providers)
- No custom JS — only htmx for interactivity
- Tailwind v4 via npm (not CDN), htmx via npm
- No remote resources (Google Fonts, CDNs) — fonts via @fontsource, served locally

## Git

- Auto-commit after logical steps
- Short descriptive messages, no descriptions
- No co-authored-by, no push
