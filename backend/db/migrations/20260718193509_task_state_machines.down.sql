DROP INDEX IF EXISTS idx_task_state_transitions_history;
DROP TABLE IF EXISTS task_state_transitions;

ALTER TABLE tasks
    DROP CONSTRAINT IF EXISTS ck_tasks_financial_state,
    DROP CONSTRAINT IF EXISTS ck_tasks_work_state,
    DROP COLUMN IF EXISTS financial_state_version,
    DROP COLUMN IF EXISTS work_state_version;
