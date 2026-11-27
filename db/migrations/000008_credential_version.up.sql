BEGIN;

ALTER TABLE account
    ADD COLUMN credential_version bigint NOT NULL DEFAULT 1;

ALTER TABLE auth_session
    ADD COLUMN credential_version bigint NOT NULL DEFAULT 1;

ALTER TABLE auth_session
    ALTER COLUMN credential_version DROP DEFAULT;

COMMIT;
