# CI and release workflows

What runs in GitHub Actions, what each workflow publishes, and how it behaves on a fork.

## Workflows

| Workflow | Trigger | What it does |
|---|---|---|
| `go.yml` (CI) | push/PR to `main` and `dev` | gofmt check, build, `go vet`, the full suite under `-race` with coverage, the coverage badge gist (upstream `main` only) and the coverage gate. |
| `codeql-analysis.yml` | push/PR to `main`, weekly | CodeQL static analysis for Go. |
| `docker-image-beta.yml` | hourly (`23 * * * *`), manual | Gate job decides, then calls `docker-build.yml` to push `beta` and `beta-<short sha>`. See [Beta images](#beta-images). |
| `docker-image.yml` | release published | Calls `docker-build.yml` to push the release tag and `latest`. |
| `docker-build.yml` | called by the two above | The shared multi-arch image build (`linux/amd64`, `arm64`, `arm/v7`, `386`). Replaces the `{{RELEASE_TAG}}` placeholder in `config/config.go` with the version it's given. |
| `release.yaml` | release published | Cross-compiles the binaries with `wangyoucao577/go-release-action` and attaches them to the release. |

## Multi-arch image builds

The `Dockerfile` builder stage is `FROM --platform=$BUILDPLATFORM`: it runs natively on the
runner, and Go cross-compiles for each target from `TARGETOS`/`TARGETARCH` (`GOARM` from
`TARGETVARIANT`, only for `GOARCH=arm`). Only the runtime stage's single `RUN apk add` runs
under QEMU, so `docker-build.yml` sets up QEMU for `arm64` and `arm` only (`386` runs
natively on an amd64 runner). Keep it to one `RUN` in that stage; use `COPY --chmod` rather
than a separate `chmod` step.

The build uses the GitHub Actions layer cache (`type=gha`, one scope shared by beta and
release builds), which mainly saves the `go mod download` layer.

Building the image needs BuildKit (`docker buildx`). Building for arm locally also needs QEMU
registered on the host (`docker run --privileged --rm tonistiigi/binfmt --install arm64,arm`).

## Beta images

`docker-image-beta.yml` doesn't run on every push to `main`. It runs every hour at :23 and
on manual dispatch, never two at once (`concurrency: beta-image`, no cancelling).

A cheap **gate** job compares `main`'s HEAD with the `org.opencontainers.image.revision`
label of `ghcr.io/<owner>/<repo>:beta`, and the build runs only if all of these hold:

- the label is missing or unreadable, or differs from HEAD (unchanged: skip);
- something besides `docs/**`, `*.md` and `.github/ISSUE_TEMPLATE/**` changed since then
  (the published revision not being in history counts as a change);
- every push-triggered workflow run for HEAD (`go.yml`, CodeQL, ...) finished green, and
  `go.yml` is among them. A running or red `main` holds the beta back until the next hour; it
  never falls back to an older green commit.

The reason is written to the run's step summary, and a skipped hour ends green.

The build checks out exactly the gated commit and pushes two tags: `beta` (moves) and the
immutable `beta-<short sha>`, the same string as the in-app version. To roll back, pin
`image: ghcr.io/aunefyren/poenskelisten:beta-<sha>`.

## Forks

The workflows need no extra secrets on a fork:

- **GHCR** logins use the built-in `GITHUB_TOKEN` (the jobs request `packages: write`), and
  the image goes to `ghcr.io/<fork owner>/<repo>`, lowercased.
- **Docker Hub** is optional. It's used only when both `DOCKER_USERNAME` and
  `DOCKER_PASSWORD` secrets exist, and the image name comes from the `DOCKERHUB_IMAGE`
  repository variable. Upstream it defaults to `aunefyren/poenskelisten`; a fork with the
  secrets but no variable publishes to GHCR only.
- **Release binaries** upload with `GITHUB_TOKEN` (`contents: write`).
- GitHub disables scheduled workflows on forks until Actions is enabled there, so a fork's
  beta runs from manual dispatch until then.
- The beta gate only ever builds `main`. A fork that wants an image per push from another
  branch can add its own workflow (e.g. `on: push` to that branch) that calls
  `docker-build.yml` with its own `ref`, `version` and `tags`. Forks build on those inputs
  (`ref`, `version`, `tags`, `latest`), so keep them stable.

The coverage badge step in `go.yml` still uses the maintainer's `GIT_HUB_TOKEN` secret (a
gist write), and is skipped when it isn't set.
