CREATE TABLE trustee_assignments (
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    trustee_index INTEGER NOT NULL CHECK (trustee_index BETWEEN 0 AND 4),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL CHECK (status IN ('invited', 'accepted', 'active', 'revoked', 'replaced')),
    nominated_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    nomination_message VARCHAR(500) NOT NULL DEFAULT '',
    proof_challenge VARCHAR(128),
    invited_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    ended_by BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    end_reason VARCHAR(500),
    PRIMARY KEY (task_slug, trustee_index),
    CONSTRAINT uq_trustee_assignment_user UNIQUE (task_slug, user_id),
    CONSTRAINT ck_trustee_assignment_acceptance CHECK (
        (status = 'invited' AND accepted_at IS NULL) OR
        (status <> 'invited' AND accepted_at IS NOT NULL)
    ),
    CONSTRAINT ck_trustee_assignment_activation CHECK (
        (status IN ('invited', 'accepted') AND activated_at IS NULL) OR
        (status IN ('active', 'revoked', 'replaced') AND activated_at IS NOT NULL)
    ),
    CONSTRAINT ck_trustee_assignment_ending CHECK (
        (status IN ('invited', 'accepted', 'active') AND ended_at IS NULL AND ended_by IS NULL AND end_reason IS NULL) OR
        (status IN ('revoked', 'replaced') AND ended_at IS NOT NULL AND ended_by IS NOT NULL AND end_reason IS NOT NULL)
    )
);

CREATE INDEX idx_trustee_assignments_user
    ON trustee_assignments (user_id, status, invited_at DESC);

CREATE TABLE trustee_assignment_history (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    trustee_index INTEGER NOT NULL CHECK (trustee_index BETWEEN 0 AND 4),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(32) NOT NULL CHECK (action IN ('nominated', 'accepted', 'activated', 'revoked', 'replaced')),
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trustee_assignment_history_task
    ON trustee_assignment_history (task_slug, created_at DESC, id DESC);

CREATE TABLE trustee_key_versions (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL,
    trustee_index INTEGER NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    version INTEGER NOT NULL CHECK (version > 0),
    xpub VARCHAR(255) NOT NULL,
    web_crypto_pubkey_hex VARCHAR(512) NOT NULL,
    xpub_proof_signature_hex VARCHAR(160) NOT NULL,
    web_crypto_proof_signature_hex VARCHAR(256) NOT NULL,
    proof_challenge VARCHAR(128) NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ,
    rotated_by BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    rotation_reason VARCHAR(500),
    FOREIGN KEY (task_slug, trustee_index)
        REFERENCES trustee_assignments(task_slug, trustee_index) ON DELETE CASCADE,
    CONSTRAINT uq_trustee_key_version UNIQUE (task_slug, trustee_index, version),
    CONSTRAINT ck_trustee_key_rotation CHECK (
        (revoked_at IS NULL AND rotated_by IS NULL AND rotation_reason IS NULL) OR
        (revoked_at IS NOT NULL AND rotated_by IS NOT NULL AND rotation_reason IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_active_trustee_xpub
    ON trustee_key_versions (xpub) WHERE revoked_at IS NULL;

CREATE UNIQUE INDEX uq_active_trustee_web_crypto_key
    ON trustee_key_versions (web_crypto_pubkey_hex) WHERE revoked_at IS NULL;

-- Existing rows predate invitations. Preserve them as active assignments and
-- version-one keys so deployments can migrate without silently dropping a
-- relationship that currently authorizes task verification.
INSERT INTO trustee_assignments (
    task_slug, trustee_index, user_id, status, nominated_by,
    nomination_message, accepted_at, activated_at
)
SELECT task_slug, trustee_index, user_id, 'active',
       (SELECT creator_id FROM tasks WHERE tasks.slug = trustee_keys.task_slug),
       'migrated from pre-onboarding trustee registration', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM trustee_keys;

INSERT INTO trustee_assignment_history (
    task_slug, trustee_index, user_id, action, actor_user_id, reason
)
SELECT a.task_slug, a.trustee_index, a.user_id, 'activated', a.nominated_by,
       'migrated from pre-onboarding trustee registration'
FROM trustee_assignments a;

INSERT INTO trustee_key_versions (
    task_slug, trustee_index, user_id, version, xpub,
    web_crypto_pubkey_hex, xpub_proof_signature_hex,
    web_crypto_proof_signature_hex, proof_challenge
)
SELECT task_slug, trustee_index, user_id, 1, xpub,
       web_crypto_pubkey_hex, 'legacy-unverified', 'legacy-unverified', 'legacy-unverified'
FROM trustee_keys;

-- Invitations must also participate in the trustee/volunteer exclusion. Waiting
-- until activation would let a volunteer accept a position they can never use.
CREATE FUNCTION reject_assignment_volunteer_conflict()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM tasks
		WHERE slug = NEW.task_slug AND creator_id = NEW.user_id
	) THEN
		RAISE EXCEPTION 'a task creator cannot be a trustee on their own task'
			USING ERRCODE = '23514';
	END IF;
    IF EXISTS (
        SELECT 1 FROM task_applications
        WHERE task_slug = NEW.task_slug AND volunteer_id = NEW.user_id
    ) THEN
        RAISE EXCEPTION 'a task volunteer cannot be nominated as a trustee on that task'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trustee_assignments_reject_volunteers
BEFORE INSERT OR UPDATE OF task_slug, user_id
ON trustee_assignments
FOR EACH ROW
EXECUTE FUNCTION reject_assignment_volunteer_conflict();

CREATE OR REPLACE FUNCTION reject_trustee_volunteer_conflict()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM trustee_assignments
        WHERE task_slug = NEW.task_slug
          AND user_id = NEW.volunteer_id
          AND status IN ('invited', 'accepted', 'active')
    ) THEN
        RAISE EXCEPTION 'a task trustee cannot also be a volunteer on that task'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
