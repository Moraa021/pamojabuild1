# Developer Notes

These notes are for repeated review while building PamojaBuild. They are intentionally beginner-friendly and should only grow when there is a useful explanation, an important project decision, or architecture context worth remembering.

## Suggested Docs Structure

A healthy docs layout separates stable engineering references from temporary working notes.

Suggested structure:

```text
backend/docs/
  architecture/
    architecture_v1.md
    security_model.md
    data_model.md
    event_model.md

  api/
    openapi.yaml
    api_contract_notes.md

  operations/
    local_dev.md
    environment_variables.md
    deployment_checklist.md
    runbooks.md

  phases/
    current_state.md
    phase_5_lightning_plan.md
    phase_6_escrow_plan.md

  decisions/
    ADR-0001-lightning-node-boundary.md
    ADR-0002-event-bus-choice.md

  dev/
    current_state.md
    lightning_impl_plan.md
    skeleton.md
    workflow.md
    dev_notes.md
```

How to think about each folder:

- `architecture/`: stable system design docs that other developers should trust.
- `api/`: frontend/backend contract docs, including OpenAPI or Swagger output.
- `operations/`: how to configure, run, deploy, monitor, and recover the system.
- `phases/`: implementation plans and progress tracking.
- `decisions/`: short records explaining why important choices were made.
- `dev/`: working notes, scaffolds, drafts, and learning material that may change often.

The current structure is fine while the project is young:

```text
backend/docs/
  architecture_v1.md
  events.md
  dev/
    current_state.md
    lightning_impl_plan.md
    skeleton.md
    workflow.md
    dev_notes.md
```

Over time, move docs out of `dev/` when they become stable source-of-truth references.

## Step 4: Split Fake Lightning From Real Lightning

Before step 4, one piece of code was doing two different jobs:

1. Pretending to be a Lightning node by making fake invoices.
2. Saving invoice records in the database.

That is risky because a Lightning node and a database are very different systems. They fail differently, need different tests, and should not be hidden behind one large interface.

The code now separates those responsibilities:

- `NodeClient`: talks to Lightning/LND.
- `Repository`: reads and writes PamojaBuild's `lightning_invoices` table.

The service flow is now:

```text
Donor asks for invoice
      ↓
Lightning service asks the NodeClient for an invoice
      ↓
NodeClient talks to LND in production, or a fake node in tests
      ↓
Lightning service validates the invoice
      ↓
Repository saves the invoice in the database
      ↓
API returns the invoice to the donor
```

The main architecture decision is that fake Lightning belongs in a fake `NodeClient`, not inside the database repository. The repository should only persist invoice records.

## LND Environment Variables

`LND_CLIENT_MODE` chooses which real LND adapter the backend uses.

For production work, the default is:

```bash
LND_CLIENT_MODE=grpc
```

gRPC is the default because LND's invoice settlement updates are a long-running stream, and gRPC handles typed streams more naturally than REST. The REST client still exists as a fallback/reference path:

```bash
LND_CLIENT_MODE=rest
```

`LND_HOST` is the address of the LND gRPC API.

Example:

```bash
LND_HOST=localhost:10009
```

This is the main host setting when `LND_CLIENT_MODE=grpc`.

LND's official release tags currently include `beta` in the LND version name, for example `v0.20.2-beta`. That is different from using a beta Go compiler. For Go itself, stable versions look like `go1.25.5`; pre-release versions include labels such as `beta` or `rc`.

The backend mirrors LND's protobuf replacement:

```go
replace google.golang.org/protobuf => github.com/lightninglabs/protobuf-go-hex-display v1.33.0-hex-display
```

This is needed because LND uses protobuf JSON behavior from that replacement, and Go applications do not automatically inherit `replace` rules from dependencies.

`LND_REST_HOST` is the address of the LND REST API.

Example:

```sh
LND_REST_HOST=https://localhost:8080
```

It tells the backend where to send REST requests when `LND_CLIENT_MODE=rest`.

`LND_MACAROON` or `LND_MACAROON_HEX` is how the backend proves to LND that it is allowed to make requests.

Examples:

```sh
LND_MACAROON=/path/to/admin.macaroon
```

or:

```sh
LND_MACAROON_HEX=0201036c6e6402f8...
```

Use one or the other. Treat macaroons like sensitive credentials. A powerful macaroon can authorize sensitive Lightning node actions.

`LND_TLS_PATH` points to LND's TLS certificate file.

Example:

```sh
LND_TLS_PATH=/path/to/tls.cert
```

It lets the backend verify that it is really talking to the intended LND node.

Short version:

- `LND_REST_HOST`: where LND is.
- `LND_CLIENT_MODE`: whether the backend uses the gRPC or REST LND adapter.
- `LND_HOST`: where LND's gRPC server is.
- `LND_REST_HOST`: where LND's REST server is, only for REST mode.
- `LND_MACAROON` or `LND_MACAROON_HEX`: proof that the backend is allowed to talk to LND.
- `LND_TLS_PATH`: proof that the server reached is really the intended LND server.

## Swagger / OpenAPI Notes

Some handlers have comments like:

```go
// Register godoc
// @Summary      Register a new user
// @Description  Create a user account and start an HttpOnly cookie session.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "Registration payload"
// @Success      201   {object}  AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Router       /api/v1/auth/register [post]
```

These comments are used by Swagger/OpenAPI tools to generate API documentation.

Swagger/OpenAPI should document:

- The endpoint path and HTTP method.
- What the endpoint does.
- Request body shape.
- Required path/query parameters.
- Success response shape.
- Error response shape.
- Authentication requirements.
- Tags that group related endpoints.

Swagger comments should describe the API contract, not internal implementation details. They help frontend developers, testers, and future backend developers understand how to call the API correctly.
