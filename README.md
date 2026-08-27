# botdo

botdo is a private operations board for AI coding agents. Add work from a
browser, let an agent claim it through the REST API, and watch status and logs
from one place.

It runs as a single Go binary with an embedded web UI and a JSON data store.
The hosted configuration adds token-protected workspaces and an optional paid
plan link without requiring a third-party auth SDK.

## Run locally

```bash
go run . --no-agent
```

Open <http://localhost:8080>. To let botdo execute a locally installed Claude
Code CLI, omit `--no-agent` and set `--workspace` to the repository it may use.

## Hosted configuration

```bash
export BOTDO_API_KEY="$(openssl rand -hex 24)"
export BOTDO_CHECKOUT_URL="https://your-payment-provider.example/checkout"
export BOTDO_DATA="/data/botdo.json"
export PORT=8080
go run . --no-agent
```

| Variable | Purpose |
| --- | --- |
| `BOTDO_API_KEY` | Protects workspace data. Leave empty only for local use. |
| `BOTDO_CHECKOUT_URL` | Shows the `$29/mo` upgrade link in the dashboard. |
| `BOTDO_DATA` | Persistent JSON data path; defaults to `botdo.json`. |
| `BOTDO_ADDR` | Listen address. `PORT` is also supported. |

The browser exchanges the workspace token for a 30-day, HttpOnly, SameSite
cookie. API clients can instead send either:

```text
Authorization: Bearer <BOTDO_API_KEY>
X-Botdo-Token: <BOTDO_API_KEY>
```

Do not expose agent execution on a public deployment. The container starts
with `--no-agent`; customers should run their agent worker in the environment
that contains their source code and credentials.

## Container

```bash
docker build -t botdo .
docker run --rm -p 8080:8080 \
  -e BOTDO_API_KEY=replace-with-a-long-random-value \
  -v botdo-data:/data \
  botdo
```

The service exposes `GET /healthz` for platform health checks. Mount `/data` on
persistent storage or task data will be lost when the container is replaced.

## API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET`, `POST` | `/api/tasks` | List or create tasks |
| `GET`, `PUT`, `DELETE` | `/api/tasks/{id}` | Read, update, or delete a task |
| `POST` | `/api/tasks/{id}/claim` | Move a pending task to in progress |
| `POST` | `/api/tasks/{id}/complete` | Complete with `done` or `failed` |
| `GET` | `/api/agents/{agent}/tasks` | Poll active work for an agent |
| `GET`, `POST` | `/api/projects` | List or create spaces |
| `GET` | `/api/tasks/{id}/logs` | Read logs; `?stream=1` uses SSE |

## Development

```bash
go test ./...
go vet ./...
go build -o botdo .
```

See [LAUNCH.md](LAUNCH.md) for the initial offer, launch checklist, and the
standard required before revenue can be claimed.
