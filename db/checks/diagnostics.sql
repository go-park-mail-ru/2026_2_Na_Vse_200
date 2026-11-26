SHOW config_file;
SHOW shared_preload_libraries;

SELECT name, setting, unit, sourcefile
FROM pg_settings
WHERE name IN ('logging_collector', 'log_destination', 'log_directory',
               'log_min_duration_statement', 'log_rotation_age', 'log_rotation_size',
               'log_parameter_max_length', 'log_parameter_max_length_on_error')
ORDER BY name;

BEGIN;
LOAD 'auto_explain';
SET LOCAL auto_explain.log_min_duration = '500ms';
SET LOCAL auto_explain.log_analyze = off;
SET LOCAL auto_explain.log_parameter_max_length = 0;
SET LOCAL ROLE music_api;
SELECT pg_sleep(0.65) AS dz3_slow_check;
ROLLBACK;

\set ON_ERROR_STOP off
SELECT 1 / 0;
\if :ERROR
    SELECT :'SQLSTATE' = '22012' AS expected_error \gset
\else
    \set expected_error false
\endif
\set ON_ERROR_STOP on
\if :expected_error
    \echo SQLSTATE 22012 recorded
\else
    DO $$ BEGIN RAISE EXCEPTION 'expected division_by_zero (22012)'; END $$;
\endif
