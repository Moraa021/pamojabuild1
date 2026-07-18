DROP TRIGGER IF EXISTS trustee_keys_reject_volunteers ON trustee_keys;
DROP FUNCTION IF EXISTS reject_volunteer_trustee_conflict();
DROP TRIGGER IF EXISTS task_applications_reject_trustees ON task_applications;
DROP FUNCTION IF EXISTS reject_trustee_volunteer_conflict();

ALTER TABLE users
    DROP COLUMN token_version,
    DROP COLUMN is_admin,
    ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'volunteer';

ALTER TABLE users
    RENAME COLUMN phone_number TO email;
