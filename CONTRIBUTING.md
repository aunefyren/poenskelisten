# Contributing to Pønskelisten

Thanks for wanting to help. Bug reports, feature ideas and pull requests are all welcome.

## Before you start

- **Small fixes** (a bug, a typo, a missing test): just open a pull request.
- **Bigger changes** (a new feature, a new endpoint or MCP tool, a schema change, anything touching auth or claims): open an issue first and describe the shape of the change. It's easier to agree on the approach before code exists than to rework a finished PR.
- Check [`docs/wip.md`](docs/wip.md). It lists known bugs, intentional coverage gaps and planned features, so you can see whether something is already known or planned differently.

## Pull requests

- Target the **`main`** branch.
- Keep each PR to **one intention**. If a feature needs a refactor first, send the refactor as its own PR (or at least its own commit).
- Start the PR title with a prefix:
  - `feature:` new behavior
  - `fix:` a bug fix
  - `docs:` documentation only
  - `ci:` GitHub Actions workflows and the Docker build
  - `maintenance:` dependencies, refactors, anything else
- Link the issue the PR addresses, if there is one.
- Update the docs your change affects: `README.md` for anything users or admins see (config options, MCP tools, features), and `docs/wip.md` if your change fixes or adds an entry there.

## Code

Read [`docs/development.md`](docs/development.md) before writing code. It covers the conventions this repo expects, most importantly:

- **Layering is strict:** `controllers` → `database` → `models`. Controllers never import `gorm.io/*` or touch `database.Instance`.
- Handler errors follow a fixed shape (log, respond, abort, return), and "not found" stays distinguishable from "query failed".
- camelCase identifiers, with acronyms in one case (`userID`, not `userId`).
- Comments explain *why*, not *what*.

`CLAUDE.md` has an architecture overview (route groups, the three token types, how the frontend is served) that is useful even if you don't use an AI assistant.

## Tests

Tests sit next to the code (`foo.go` → `foo_test.go`, same package). New behavior needs tests, and bug fixes should come with a test that would have caught the bug. `docs/development.md` describes the testing conventions and the shared fixtures in `controllers/helpers_test.go`.

## Before you push

CI runs these, and all of them must pass:

```
gofmt -l .        # must print nothing; gofmt -w . to fix
go build ./...
go vet ./...
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 20m ./...
```

CI also fails if total test coverage falls below its minimum (see the "Coverage gate" step in `.github/workflows/go.yml`). Check your total with `go tool cover -func=coverage.out | tail -1`.

## Reporting bugs

Open an issue with:

- your Pønskelisten version
- how you run it (Docker or binary)
- your database (SQLite, PostgreSQL or MySQL)
- the steps to reproduce the bug
- any relevant log output

Report security vulnerabilities privately through GitHub's "Report a vulnerability" button on the repository's Security tab rather than in a public issue.
