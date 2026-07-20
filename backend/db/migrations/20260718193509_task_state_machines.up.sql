-- Normalize the unsupported status previously written by the submission event
-- subscriber before constraining the work-state vocabulary.
UPDATE tasks
SET status = 'pending_verification'
WHERE status = 'submitted';

ALTER TABLE tasks
    ADD COLUMN work_state_version BIGINT NOT NULL DEFAULT 1
        CHECK (work_state_version > 0),
    ADD COLUMN financial_state_version BIGINT NOT NULL DEFAULT 1
        CHECK (financial_state_version > 0),
    ADD CONSTRAINT ck_tasks_work_state
        CHECK (status IN ('open', 'in_progress', 'pending_verification', 'completed')),
    ADD CONSTRAINT ck_tasks_financial_state
        CHECK (financial_state IN (
            'ACTIVE',
            'LIQUIDATING',
            'READY_FOR_PAYOUT',
            'PAYOUT_PROCESSING',
            'ARCHIVED',
            'SYSTEM_LOCKDOWN'
        ));

CREATE TABLE task_state_transitions (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE RESTRICT,
    state_kind VARCHAR(16) NOT NULL
        CHECK (state_kind IN ('work', 'financial')),
    from_state VARCHAR(50),
    to_state VARCHAR(50) NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    actor_user_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    reason VARCHAR(500) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_task_state_transition_version
        UNIQUE (task_slug, state_kind, version),
    CONSTRAINT uq_task_state_transition_idempotency
        UNIQUE (task_slug, state_kind, idempotency_key),
    CONSTRAINT ck_task_state_transition_from_state
        CHECK (from_state IS NULL OR from_state <> to_state),
    CONSTRAINT ck_task_state_transition_value
        CHECK (
            (state_kind = 'work' AND to_state IN (
                'open',
                'in_progress',
                'pending_verification',
                'completed'
            ))
            OR
            (state_kind = 'financial' AND to_state IN (
                'ACTIVE',
                'LIQUIDATING',
                'READY_FOR_PAYOUT',
                'PAYOUT_PROCESSING',
                'ARCHIVED',
                'SYSTEM_LOCKDOWN'
            ))
        )
);

CREATE INDEX idx_task_state_transitions_history
    ON task_state_transitions (task_slug, created_at DESC, id DESC);

-- Existing rows receive an auditable baseline. NULL from_state distinguishes
-- migration baselines and task creation from real lifecycle transitions.
INSERT INTO task_state_transitions (
    task_slug,
    state_kind,
    from_state,
    to_state,
    version,
    actor_user_id,
    reason,
    idempotency_key
)
SELECT
    slug,
    'work',
    NULL,
    status,
    work_state_version,
    creator_id,
    'state machine migration baseline',
    'migration-baseline-work'
FROM tasks;

INSERT INTO task_state_transitions (
    task_slug,
    state_kind,
    from_state,
    to_state,
    version,
    actor_user_id,
    reason,
    idempotency_key
)
SELECT
    slug,
    'financial',
    NULL,
    financial_state,
    financial_state_version,
    creator_id,
    'state machine migration baseline',
    'migration-baseline-financial'
FROM tasks;
