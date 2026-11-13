# Development and verification

Run these commands from the repository root unless a module directory is
specified. Use the Go toolchain declared in `go.mod` and the repository's
`mise.toml` tasks. The committed `go.work` connects five modules; a root
`go test ./...` does not test nested modules.

- [Local checks and push gate](#local-checks-and-push-gate)
- [Runtime pins and container checks](#runtime-pins-and-container-checks)
- [Documentation and evidence maintenance](#documentation-and-evidence-maintenance)

## Local checks and push gate

The repository root provides verification tasks across all five modules
and the currently pinned target runtimes:

```sh
mise run lint               # all five Go modules
mise run check-mod-tidy     # fail if go mod tidy would change module files
mise run govulncheck        # all five Go modules
mise run pre-push           # tidy + standalone builds + lint + govulncheck
mise run verify-local       # modules, generated evidence, OKF, Markdown policy
mise run verify-containers  # exact Emulator/Omni manifests; no evidence writes
mise run verify-managed     # point-in-time checks against TEST_REAL_SPANNER_DATABASE
mise run verify-current     # local + containers + managed
```

`check-mod-tidy` runs `go mod tidy` in each module and fails if module files
change; it can leave those changes in the checkout for inspection.

Activate the versioned pre-push hook once in each clone:

```sh
mise run hooks-install
```

The hook runs `mise run pre-push`, the same tidy, standalone-build, lint, and
govulncheck gates as CI, and blocks a push when any of them fail. Git does not activate
repository hooks on clone, so the installation step is intentionally explicit.
`git push --no-verify` can still bypass a local hook; CI remains the
authoritative shared gate.

## Runtime pins and container checks

When using an active Colima Docker context, pass both the host socket and the
container-side socket to testcontainers. A working `docker info` alone does not
ensure that testcontainers can discover the daemon or mount its cleanup socket:

```sh
DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')" \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock \
mise run verify-containers
```

[`runtime_targets.json`](runtime_targets.json) is the single source of truth
for the descriptive tags and platform-specific OCI manifest digests used by
Emulator and Omni tests. Container check commands compare live catalog surfaces
with retained evidence and emit compact JSON; they do not create new captures.
The two Omni plan suites share one runtime in `verify-containers`, avoiding a
second cold container startup.

The current defaults are Emulator `1.5.59` and Omni `2026.r4-lts.2`. The
three container-consuming modules use spanemuboost `v0.4.10`, whose
Omni startup support remains experimental. Omni `2026.r5.1` is an explicitly
selectable alternative in the same registry, with its own platform digests:

```sh
mise run verify-omni-integration
mise run verify-omni-r5-integration
# Equivalent explicit selection from the tools module:
go run ./omni-integration-runner --repo-root .. --omni-version 2026.r5.1
```

Selecting the alternative does not change the default. Unknown version names
fail before a container is started. Each report records the selected tagged
manifest reference; an available image tag alone is not runtime verification.

These are the checked-in selections, not a live release-discovery result.
`verify-managed` requires `TEST_REAL_SPANNER_DATABASE` and configured credentials;
its observations are point-in-time. New disposable-database investigations need
their own authorized lifecycle and are not implied by editing documentation.

## Documentation and evidence maintenance

Keep library guidance in [godoc](doc.go) and runnable Go Example tests, analyzer
command usage in the [command guide](cmd/spanner-analyzer/README.md), and query
generator UX in [its guide](cmd/spanner-query-gen/README.md). The root README is
the starting point, not a second detailed manual.

Use the [documentation authority guide](knowledge/concepts/documentation-authority.md)
to distinguish implemented behavior, intended design, backlog, and evidence.
New research belongs under `knowledge/research/`; the legacy `research/` tree
remains historical. Keep engine observations in their
[chronological histories](knowledge/research/runtime-verification/index.md).
Retain reproduction inputs and compact measured results; routine full logs
belong in task scratch or CI artifacts. See the
[evidence lifecycle](knowledge/concepts/evidence-lifecycle.md#minimum-retained-evidence).

When adding or removing Markdown outside the bundle, update the
[repository-document inventory](knowledge/references/repository-documents.md).
Bundle concepts also need metadata, directory reachability, and publication
inventory membership as described in [AGENTS.md](AGENTS.md). Run:

```sh
(cd tools && go run ./okf-check --repo-root .. --gate all)
(cd tools && go run ./markdown-language-check --repo-root ..)
```

These checks do not publish to Google Cloud. Use `--publication-require-clean`
for a committed bundle before any separately authorized synchronization.
