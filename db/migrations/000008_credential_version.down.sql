BEGIN;

DELETE FROM auth_session AS session
USING account
WHERE account.id = session.account_id
  AND account.credential_version <> session.credential_version;

ALTER TABLE auth_session DROP COLUMN credential_version;
ALTER TABLE account DROP COLUMN credential_version;

COMMIT;
