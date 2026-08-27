# Cachicamas

<p align="center">
  <img src="docs/assets/cachicamas-logo.png" alt="Cachicamas" width="240" height="240" />
</p>

Cachicamas is a multiplayer agentic system for building and running a company. It gives teams specialist agents for work such as software development, database administration, ticketing, finance, and marketing—and a shared runtime where those agents can collaborate.

The project is pre-release. [Witsaba](https://witsaba.com/) is its first user, but the platform is designed for any company.

## How it works

Cachicamas separates reusable agent mechanics from business-specific behavior:

1. **Model adapter** — a vendor-portable interface to language models.
2. **Agent runtime** — the model loop, tool execution, permissions, and conversation lifecycle shared by every agent.
3. **Archetypes** — specialist agents that add their own policy, tools, resources, persistence, and interface.

The coding archetype is the first archetype. Other business systems expose their capabilities over the network, with one owning archetype per system. This keeps the runtime independent of any single role or vendor.

For the complete design, read the [agent stack architecture](docs/architecture/0001-cachicamas-agent-stack-v2.md) and [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md).

## Architecture

| Component | Responsibility |
| --- | --- |
| `backend/agent` | Layered Go module containing the model adapter, portable agent runtime, archetypes, and chat service. |
| `backend/database_administrator` | Hexagonal Go API for identity, organizations, workspaces, sync jobs, prompts, skills, and database migrations. |
| `backend/workspace_syncer` | Internal Go service that clones and validates Git workspaces. |
| `frontend` | Qwik operator interface for authentication, workspaces, agent chat, prompts, and skills. |
| PostgreSQL | Persistent storage, with each business system responsible for its own data. |
| OpenTelemetry + Jaeger | Local telemetry collection and trace exploration. |

The business services follow a hexagonal layout:

```text
src/
├── application/   # use cases
├── domain/        # entities and contracts; no I/O
├── interfaces/    # inbound interfaces, including HTTP
├── infrastructure/ # outbound adapters, including PostgreSQL
├── migration/     # database migrations where applicable
├── otel/          # observability wiring
└── cmd/           # service entry points
```

`backend/agent` intentionally uses a layered architecture instead. Archetypes communicate with business systems over network contracts rather than importing their Go modules.

## Quick start

### Prerequisites

- Docker with Compose v2
- Go 1.26.6 or later for backend development
- Node.js 18.17 or later and pnpm 11.8 or later for frontend development
- GitHub OAuth credentials and an OpenRouter API key for the complete stack

### Run the stack

```bash
git clone https://github.com/witsaba/cachicamas.git
cd cachicamas
cp .env.example .env
```

Edit `.env` and replace the placeholder credentials. At minimum, configure the OpenRouter key, GitHub OAuth credentials, and generated secrets documented in the file. Then start the services:

```bash
docker compose up -d --build
docker compose ps
```

The default local endpoints are:

| Service | URL |
| --- | --- |
| Frontend | <http://localhost:3015> |
| Database Administrator API | <http://localhost:8080> |
| Agent chat API | <http://localhost:8090> |
| Jaeger | <http://localhost:16686> |

Stop the stack with `docker compose down`.

## Development

Each backend service is an independent Go module. Run commands from the module you are changing:

```bash
cd backend/agent # or backend/database_administrator, backend/workspace_syncer
make test
make lint
make build
```

Useful backend targets include `make help`, `make fmt`, `make vet`, `make test/cover`, and `make vuln-check`. The database administrator also provides `make test/integration` for tests backed by Compose PostgreSQL.

Run the frontend separately when working on the UI:

```bash
cd frontend
pnpm install
pnpm dev
```

Before submitting frontend changes, run:

```bash
pnpm verify
```

## Engineering workflow

- Use strict TDD: red, green, then refactor.
- Keep tests beside the code they exercise.
- Use conventional commits without AI attribution or `Co-Authored-By` trailers.
- Add new top-level dependencies only with an Architecture Decision Record.
- Run source-mutating formatters before verification.

Cachicamas uses Spec-Driven Development for substantial changes. Project context and active change artifacts live in [`openspec/`](openspec/); architecture decisions and milestone plans live in [`docs/`](docs/).

## Repository map

```text
backend/       Go services and the agent stack
frontend/      Qwik application
docs/          architecture, ADRs, product requirements, and assets
infra/         PostgreSQL, OpenTelemetry, and Jaeger configuration
openspec/      specifications and active change artifacts
scripts/       development utilities
spikes/        disposable technical experiments
```

## Where to go next

- Product identity and boundaries: [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)
- Agent architecture: [Cachicamas agent stack v2](docs/architecture/0001-cachicamas-agent-stack-v2.md)
- Project conventions: [`openspec/project.md`](openspec/project.md)
- Current implementation plans: [`docs/architecture/milestones/`](docs/architecture/milestones/)

## License

See [LICENSE](LICENSE).
