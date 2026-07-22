CREATE OR REPLACE FUNCTION reject_trustee_volunteer_conflict()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM trustee_keys
        WHERE task_slug = NEW.task_slug AND user_id = NEW.volunteer_id
    ) THEN
        RAISE EXCEPTION 'a task trustee cannot also be a volunteer on that task'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trustee_assignments_reject_volunteers ON trustee_assignments;
DROP FUNCTION IF EXISTS reject_assignment_volunteer_conflict();
DROP TABLE IF EXISTS trustee_key_versions;
DROP TABLE IF EXISTS trustee_assignment_history;
DROP TABLE IF EXISTS trustee_assignments;
