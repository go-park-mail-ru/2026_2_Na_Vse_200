BEGIN;
SET LOCAL application_name = 'music-check-lock-waiter';

DO $$
BEGIN
    PERFORM pg_advisory_xact_lock(205, 30);
    RAISE EXCEPTION 'expected a lock timeout; start the lock holder first';
EXCEPTION WHEN lock_not_available THEN
    RAISE NOTICE 'lock timeout (%)', SQLSTATE;
END;
$$;

ROLLBACK;
