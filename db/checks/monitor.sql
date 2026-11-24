DO $$
BEGIN
    IF current_user <> 'music_monitor' OR session_user <> 'music_monitor' THEN
        RAISE EXCEPTION 'connect as music_monitor';
    END IF;

    IF NOT pg_has_role('pg_read_all_stats', 'USAGE')
       OR pg_has_role('music_owner', 'SET')
       OR has_schema_privilege('public', 'CREATE')
       OR has_column_privilege('public.account', 'password_hash', 'SELECT')
       OR has_table_privilege('public.account', 'INSERT, UPDATE, DELETE') THEN
        RAISE EXCEPTION 'unexpected monitoring privileges';
    END IF;

    BEGIN
        PERFORM 1 FROM public.auth_session LIMIT 1;
        RAISE EXCEPTION 'music_monitor can read auth_session';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'auth_session read denied (%)', SQLSTATE;
    END;
END;
$$;

SELECT current_user, count(*) AS visible_connections
FROM pg_stat_activity WHERE datname = current_database();
