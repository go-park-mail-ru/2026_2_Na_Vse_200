SELECT calls, rows, round(total_exec_time::numeric, 2) AS total_exec_ms,
       queryid, regexp_replace(query, '[[:space:]]+', ' ', 'g') AS query
FROM db_metrics.pg_stat_statements
WHERE userid = 'music_api'::regrole
  AND dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
  AND (query LIKE '%WITH selected_tracks AS (%' OR query LIKE '%WITH selected_albums AS (%'
       OR query LIKE '%FROM artist%LIMIT $%')
ORDER BY queryid;
