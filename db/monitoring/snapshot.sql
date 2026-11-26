\pset pager off

SELECT clock_timestamp() AS sampled_at, current_database() AS database, current_user AS reader,
       current_setting('server_version') AS version;

SELECT usename, application_name, state, wait_event_type, wait_event, count(*) AS connections
FROM pg_stat_activity
WHERE datname = current_database() AND backend_type = 'client backend'
GROUP BY usename, application_name, state, wait_event_type, wait_event
ORDER BY usename, application_name, state;

SELECT pid, application_name, state, clock_timestamp() - xact_start AS transaction_age,
       CASE WHEN state = 'active' THEN clock_timestamp() - query_start END AS query_age,
       wait_event_type, wait_event
FROM pg_stat_activity
WHERE datname = current_database() AND pid <> pg_backend_pid()
  AND (xact_start < clock_timestamp() - interval '1 second' OR wait_event_type = 'Lock')
ORDER BY xact_start NULLS LAST;

SELECT l.pid, a.application_name, l.locktype, l.mode, l.waitstart,
       pg_blocking_pids(l.pid) AS blocked_by
FROM pg_locks l JOIN pg_stat_activity a ON a.pid = l.pid
WHERE a.datname = current_database() AND NOT l.granted
ORDER BY l.waitstart;

SELECT numbackends, xact_commit, xact_rollback, deadlocks, blks_read, blks_hit,
       blk_read_time, blk_write_time, tup_inserted, tup_updated, tup_deleted,
       temp_files, temp_bytes, stats_reset, pg_database_size(datname) AS database_bytes
FROM pg_stat_database
WHERE datname = current_database();

SELECT relname, seq_scan, idx_scan, n_live_tup, n_dead_tup, last_autovacuum, last_autoanalyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY relname;
