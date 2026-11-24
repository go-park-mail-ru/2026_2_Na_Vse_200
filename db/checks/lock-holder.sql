BEGIN;
SET LOCAL application_name = 'music-check-lock-holder';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(205, 30);
SELECT pg_sleep(25);
ROLLBACK;
