BEGIN;
SET LOCAL application_name = 'music-check-timeout';

DO $$
BEGIN
    PERFORM pg_sleep(6);
    RAISE EXCEPTION 'expected a statement timeout';
EXCEPTION WHEN query_canceled THEN
    RAISE NOTICE 'statement timeout (%)', SQLSTATE;
END;
$$;

ROLLBACK;
SELECT current_user;
