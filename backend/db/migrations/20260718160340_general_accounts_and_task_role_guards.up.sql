ALTER TABLE users
    RENAME COLUMN email TO phone_number;

ALTER TABLE users
    DROP COLUMN role,
    ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN token_version BIGINT NOT NULL DEFAULT 0 CHECK (token_version >= 0);

-- Task roles live in separate relationship tables. These paired triggers make
-- the trustee/volunteer separation a database invariant, so a future code path
-- cannot accidentally bypass the service-layer authorization check.
CREATE FUNCTION reject_trustee_volunteer_conflict()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM trustee_keys
        WHERE task_slug = NEW.task_slug
          AND user_id = NEW.volunteer_id
    ) THEN
        RAISE EXCEPTION 'a task trustee cannot also be a volunteer on that task'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER task_applications_reject_trustees
BEFORE INSERT OR UPDATE OF task_slug, volunteer_id
ON task_applications
FOR EACH ROW
EXECUTE FUNCTION reject_trustee_volunteer_conflict();

CREATE FUNCTION reject_volunteer_trustee_conflict()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM task_applications
        WHERE task_slug = NEW.task_slug
          AND volunteer_id = NEW.user_id
    ) THEN
        RAISE EXCEPTION 'a task volunteer cannot also be a trustee on that task'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trustee_keys_reject_volunteers
BEFORE INSERT OR UPDATE OF task_slug, user_id
ON trustee_keys
FOR EACH ROW
EXECUTE FUNCTION reject_volunteer_trustee_conflict();
