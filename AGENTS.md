# Sclera

Go web API server (Go 1.26.6, module `github.com/mattthew/sclera` — note the double `t`).

Plain `net/http` + `http.ServeMux`; templ-rendered frontend compiled into the binary.

Database: pgx/v5 → PostgreSQL.

Redis is used for rate limiting and OTP management.

Resend is used for email.

No web framework, ORM, or alternate HTTP router.

## Commands

* Build:
  `go build -o tmp/main ./cmd/server`

  * Entrypoint: `cmd/server/main.go`
  * `tmp/` is gitignored.

* Development:
  `air`

  * Reads `.air.toml`.
  * Rebuilds `./tmp/main`.
  * Runs the server on `:8080`.
  * Does **not** run `templ generate`.

* Test:
  `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`

  * This matches CI.
  * There are currently no `_test.go` files.

* Lint/security:
  `golangci-lint run ./...`
  `govulncheck ./...`

  * CI uses the latest `golangci-lint`.
  * There is currently no `.golangci.yml`, so default configuration applies.

* Install golangci-lint:
  `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

* CI order:
  `golangci-lint → govulncheck → go test → go build`

  * See `.github/workflows/ci.yml`.

## templ

templ is easy to get wrong.

* `.templ` source files are under:
  `tempFrontend/{shared,userHandling,LLMHandling}`.
* Generated `*_templ.go` files are committed.
* Generated files are served through `internal/httpcallers`.
* After modifying any `.templ` file, run:
  `templ generate ./...`
* CI does **not** regenerate templ output.
* Stale generated files can therefore pass CI silently.

Do not assume generated templ files are current.

## Architecture

### Routing

`internal/routes` registers every endpoint on the mux.

Current route registration includes:

* `RegisterUserRoutes`
* `RegisterGemmaRoutes`

`internal/server/runSever.go` constructs the mux/handler and starts the server on `:8080`.

### Request flow

Endpoints generally follow:

`routes → middleware/rate limiting → httpcaller/handler`

JWT-protected routes use:

`middleware.CheckJwtToken`

JWT authentication uses the `Authorization: Bearer …` cookie/header flow.

IP-based rate limiting uses:

`httpcallers.WithIPRateLimit`

Do not create a second implementation of authentication, rate limiting, or request handling when an existing implementation already provides the required behavior.

### Redis token costs

Token costs are defined in `internal/redisInternal`:

* Page = 5
* Form = 10
* JWT = 20
* LLM = 30

Reuse the existing rate-limit/token-cost logic rather than introducing another rate-limiting mechanism unless the task explicitly requires architectural change.

### httpcallers

`internal/httpcallers` contains both:

* templ page renderers
* server-side API handlers

Do not assume `httpcallers` contains only frontend code.

### Authentication

`internal/authentication` contains:

* JWT verification
* Redis OTP handling
* client-IP extraction

Before adding authentication-related functionality, inspect the existing implementation and its callers.

## Runtime / Docker

Only nginx is host-exposed.

Current compose networking:

* nginx:
  `127.0.0.1:81 → server:8080`
* PostgreSQL:
  `127.0.0.1:5433`
* Redis:
  `ratelimit-redis`
* SearXNG:
  `127.0.0.1:8081`
* External Docker network:
  `sclera-network`

The Go server itself is not intended to be directly exposed to the host.

### Trusted proxy

`cmd/server/main.go` currently hardcodes the trusted proxy CIDR:

`172.19.0.0/16`

This must match the Docker network subnet.

`authentication.GetClientIP` relies on this trust relationship when processing nginx-provided:

* `X-Forwarded-For`
* `X-Real-IP`

Do not change the trusted CIDR without checking the actual Docker network configuration.

## Environment

Environment variables are loaded from an untracked `.env` file through `godotenv`, with real environment variables also supported.

Expected variables:

* `DATABASE_URL`
* `JWT_SECRET`
* `POSTGRES_PASSWORD`
* `REDIS_ADDR`
* `SEARXNG_BASE_URL`

Do not commit `.env` or secrets.

## Database

Database schema is managed through migration pairs in `migrations/`.

The application uses:

* schema: `sclera`
* users table: `sclera.users`

Migrations are applied manually.

Do not modify database schema assumptions without checking the existing migrations and database access code.

## Deployment

Deployment configuration is in:

`.github/workflows/cd.yml`

CI runs for pushes to:

* `main`
* `v*` tags

CD runs after CI for pushes whose branch starts with `v`.

Version releases currently use branches named like:

`vX.Y.Z`

Deployment:

1. Build Docker image.
2. Tag image as `matthew552/sclera:<ver>`.
3. Also tag/push `latest`.
4. Push to Docker Hub.
5. Run:
   `docker compose up -d`

Check `.github/workflows/ci.yml` and `.github/workflows/cd.yml` before changing CI/CD behavior.

# Development Rules

## 1. Plan Before Building

For any non-trivial feature, refactor, architectural change, or multi-file modification:

**The first response must be planning only.**

Do not:

* write code
* modify files
* generate patches
* execute write operations

in the first response.

The first response must explicitly state:

1. What existing code and execution path the change targets.
2. Which files/functions/types are likely involved.
3. How the new behavior will integrate with the existing architecture.
4. What existing functionality will be reused.
5. What will **not** be changed.

Only after the plan is established should implementation begin.

## 2. Inspect Before Modifying

Never modify code based only on filenames, assumptions, or task descriptions.

Before changing a file:

* read the relevant implementation
* inspect the relevant callers
* inspect related types/interfaces
* inspect route registration when applicable
* inspect middleware when applicable
* inspect database/Redis interactions when applicable

Understand the current execution path before editing it.

## 3. Reuse Existing Logic

Before creating a new function, type, helper, middleware, handler, or abstraction:

1. Search the codebase for existing functionality serving the same purpose.
2. Identify adjacent implementations that could be reused or extended.
3. Prefer reusing or modifying the existing implementation when appropriate.
4. Do not create parallel implementations without explaining why.

The proposed implementation must explicitly state that it does not duplicate existing functionality.

Do not create convenience wrappers that merely rename or forward to an existing function unless there is a clear architectural reason.

## 4. Preserve the Existing Architecture

Sclera intentionally uses a small, explicit architecture.

Do not introduce the following merely for convenience:

* web frameworks
* alternate HTTP routers
* ORMs
* unnecessary service layers
* unnecessary repository layers
* dependency-injection frameworks
* large third-party abstractions
* duplicate middleware systems

Architectural changes require an explicit reason and should be discussed in the plan before implementation.

Prefer the existing standard-library-oriented design.

## 5. Minimize Scope

A feature request is not permission to refactor unrelated code.

Do not:

* rename unrelated files
* reformat unrelated packages
* reorganize the project without need
* replace working implementations unnecessarily
* upgrade dependencies without a task-related reason
* alter unrelated configuration

Make the smallest coherent change that satisfies the task.

## 6. Preserve Existing Behavior

Unless the task explicitly changes behavior, preserve:

* existing routes
* authentication semantics
* rate limiting
* Redis usage
* database semantics
* Docker networking
* environment variable names
* CI/CD behavior
* templ rendering behavior

Do not silently change behavior while implementing an unrelated feature.

## 7. templ Safety

When modifying `.templ` files:

1. Edit the `.templ` source.
2. Run `templ generate ./...`.
3. Verify the generated `*_templ.go` changes.
4. Build/test using the generated output.

Do not manually edit generated `*_templ.go` files when the source `.templ` file should be changed instead.

## 8. File Write Safety

Before any file write:

1. Review the complete pending write/edit content.
2. Verify it targets the intended file.
3. Verify the content has not been duplicated.
4. Verify the content is not accidentally concatenated twice.
5. Verify the content is not truncated.
6. Verify an edit is not replacing more of the file than intended.

For generated patches or scripted edits, inspect the resulting file after the write.

## 9. Verify Changes

After modifying code, verify the relevant behavior.

At minimum, use the smallest appropriate checks, such as:

* `go build ./...`
* `go test ./...`
* `go test -race ./...`
* `golangci-lint run ./...`
* `govulncheck ./...`

For templ changes, run `templ generate ./...` first.

For Docker/networking changes, verify the relevant compose configuration and container connectivity rather than assuming configuration is correct.

## 10. Do Not Assume Generated or Runtime State

Do not assume that:

* templ output is current
* Docker network CIDRs are unchanged
* Redis service names are unchanged
* environment variables exist
* containers are running
* migrations have been applied
* CI regenerated generated files
* a local build reflects the current source tree

Inspect the relevant configuration or runtime state when the task depends on it.

## 11. Security-Sensitive Changes

Treat changes involving these areas as security-sensitive:

* JWT
* cookies
* authentication
* authorization
* Redis-backed rate limiting
* OTP handling
* client-IP extraction
* trusted proxies
* secrets
* database access

For security-sensitive changes:

* inspect the existing flow first
* preserve existing security boundaries unless the task explicitly changes them
* avoid introducing parallel authentication or authorization logic
* do not weaken validation or trust boundaries for convenience

## 12. Agent Behavior

Do not make speculative changes "just in case."

Do not add abstractions because they might be useful later.

Do not rewrite working code merely because another style is preferred.

Do not treat a successful compilation as proof that the implementation is correct.

When uncertain, inspect the existing code and follow its established behavior rather than inventing a new pattern.

## Absolute Rules

1. **Plan first for every non-trivial task.**
2. **No code or file writes in the first response to a complex feature.**
3. **Inspect existing implementations and callers before modifying them.**
4. **Explicitly identify and reuse existing functionality before creating new functions or abstractions.**
5. **Do not duplicate existing behavior without a documented reason.**
6. **Preserve Sclera's existing architecture unless architectural change is explicitly required.**
7. **Before every file write, inspect the complete pending change for duplication, truncation, concatenation, and wrong-file targeting.**
8. **After changes, run the relevant verification commands.**
9. **Treat authentication, rate limiting, proxy trust, Redis, JWT, OTP, cookies, and secrets as security-sensitive.**
10. **Do not make unrelated changes.**