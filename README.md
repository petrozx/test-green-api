# GREEN-API test task (Go backend)

This project contains:

- frontend: `index.html`, `styles.css`, `script.js`
- backend: `main.go` (fully on Go)

Implemented methods:

- `getSettings`
- `getStateInstance`
- `sendMessage`
- `sendFileByUrl`

## Run locally

```bash
go run .
```

Open:

`http://localhost:8080`

Optional config:

- `GREEN_API_HOST` (default: `3100.api.green-api.com`)

Example:

```bash
GREEN_API_HOST=3100.api.green-api.com go run .
```

## How it works

- Browser calls local Go endpoints:
  - `POST /api/getSettings`
  - `POST /api/getStateInstance`
  - `POST /api/sendMessage`
  - `POST /api/sendFileByUrl`
- Go backend forwards requests to GREEN-API and returns JSON response to the page.

## Deploy

Deploy as a Go web app (not static hosting), for example:

- Render
- Railway
- Fly.io
- any VPS with `go run .` or built binary
