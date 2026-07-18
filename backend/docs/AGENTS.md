# PamojaBuild Coding Instructions

## Production Standard

Treat PamojaBuild as production software for real users and real funds. Do not write toy code, ignore errors, or leave unsafe placeholders in production paths.

## Security And Reliability

- Handle errors explicitly and return useful context.
- Keep money-moving logic conservative and auditable.
- Prefer clear interfaces between domains over shortcuts.
- Do not mix external infrastructure concerns with database persistence.
- Preserve idempotency for payment, settlement, ledger, and event flows.
- Avoid changes that can double-credit, double-pay, or silently lose financial state.

## Architecture

- Follow the existing domain boundaries: task, trustee, ledger, lightning, escrow, payout, audit.
- Keep database repositories focused on persistence.
- Keep external clients focused on external systems such as LND.
- Record important architecture decisions in docs when they affect future implementation.
- Use events for cross-domain communication where the architecture requires it.

## Database Standard

- PostgreSQL is the only supported application database. Do not add SQLite dependencies, SQL dialects, fallback behavior, or SQLite-backed repository tests.
- Use `golang-migrate` for versioned database migrations.
- Name migration versions with 14-digit UTC timestamps (`YYYYMMDDHHMMSS`), not team-wide sequence numbers. Use `make migration-create NAME=<description>` to create paired files.
- Every schema change must have explicit up and down migrations unless a documented, reviewed safety reason makes rollback impossible.
- Run migrations through a separate deployment command. The API server must not automatically apply migrations during startup.
- Do not build new schema work on the existing legacy SQLite-oriented migrations. Replace them with a clean PostgreSQL migration baseline during the database-standardization implementation unit.
- Test PostgreSQL-specific repositories and concurrency behavior against PostgreSQL.

## Accounts And Task Relationships

- A registered user has a general account. Creator, volunteer, and trustee are task-specific relationships rather than permanent global account roles.
- A task creator may assign themselves as a volunteer. This relationship and its payout terms must be visible before donations are accepted.
- A trustee must never also be a volunteer on the same task.
- Creators and volunteers cannot verify their own work or authorize their own payouts. Payout authorization remains subject to the task's independent trustee threshold.

## Working Documentation

- Use `backend/docs/dev/app_flow_deep_dive.md` as the main current-system reference.
- Keep `backend/docs/dev/implementation_order.md` updated as implementation units start and finish so a new chat or developer can resume without reconstructing project history.
- After each backend API or workflow change, document the frontend changes and integrations required under `backend/docs/dev`. Keep this simple, concrete, and aligned with the actual API contract.
- Clearly distinguish implemented behavior from planned behavior in documentation.

## Commits

- Commit at reasonable intervals after a cohesive implementation unit is complete, tested, and leaves the repository in a working state.
- Do not accumulate an entire phase into one large final commit.
- Use Conventional Commits, for example `feat(task): enforce financial state transitions` or `fix(ledger): prevent duplicate settlement credits`.
- Keep commits focused and exclude unrelated user changes.
- Do not describe work as complete or commit it while relevant tests are failing, unless the failure and reason are explicitly documented and accepted.

## Comments

Comment your code. But use comments to explain decisions, tradeoffs, security assumptions, and non-obvious behavior.

Good comments explain why something is done, for example:

- Why a database update must be conditional.
- Why duplicate settlements must return success without publishing another event.
- Why a boundary exists between a Lightning node client and an invoice repository.
- Why a particular validation is security-sensitive.
- Where a task, ledger, Lightning, escrow, swap, or PSBT integration boundary is important to future work.

Avoid comments that only restate obvious code, such as "this is a repository interface" or "loop over items."

## API Documentation

Handlers should use Swagger/OpenAPI comments when they expose API endpoints.

Swagger comments should document:

- Endpoint summary and description.
- Tags.
- Request body.
- Path and query parameters.
- Success responses.
- Error responses.
- Route and HTTP method.
- Authentication requirements where applicable.

Keep Swagger documentation aligned with actual handler behavior and response structs.
