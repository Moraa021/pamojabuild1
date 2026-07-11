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

## Comments

Comment your code. But use comments to explain decisions, tradeoffs, security assumptions, and non-obvious behavior.

Good comments explain why something is done, for example:

- Why a database update must be conditional.
- Why duplicate settlements must return success without publishing another event.
- Why a boundary exists between a Lightning node client and an invoice repository.
- Why a particular validation is security-sensitive.

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
