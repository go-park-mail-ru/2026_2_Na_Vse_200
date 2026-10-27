BEGIN;

CREATE TABLE auth_session (
    id uuid NOT NULL,
    account_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT auth_session_pk PRIMARY KEY (id),
    CONSTRAINT auth_session_expires_at_check CHECK (isfinite(expires_at)),
    CONSTRAINT auth_session_created_at_check CHECK (isfinite(created_at)),
    CONSTRAINT auth_session_account_fk FOREIGN KEY (account_id) REFERENCES account (id) ON UPDATE RESTRICT ON DELETE CASCADE
);

COMMIT;
