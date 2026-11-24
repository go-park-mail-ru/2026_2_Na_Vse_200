\set ON_ERROR_STOP on
\set VERBOSITY terse
\getenv migrator_password MUSIC_MIGRATOR_PASSWORD
\getenv api_password MUSIC_API_PASSWORD
\getenv monitor_password MUSIC_MONITOR_PASSWORD

BEGIN;
SET LOCAL log_min_duration_statement = -1;
SET LOCAL log_lock_waits = off;
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(205, 3);

DO $$
BEGIN
    IF current_database() <> 'music' THEN
        RAISE EXCEPTION 'roles.sql requires the music database';
    END IF;
END;
$$;

SELECT format('CREATE ROLE %I', name)
FROM (VALUES ('music_owner'), ('music_migrator'), ('music_api'), ('music_monitor')) AS roles(name)
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = roles.name)
\gexec

ALTER ROLE music_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
ALTER ROLE music_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 4;
ALTER ROLE music_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 30;
ALTER ROLE music_monitor LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 3;

GRANT music_owner TO music_migrator WITH INHERIT FALSE, SET TRUE, ADMIN FALSE;
GRANT pg_read_all_stats TO music_monitor WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;

DO $$
BEGIN
    IF EXISTS (
        SELECT FROM pg_auth_members m
        JOIN pg_roles member ON member.oid = m.member
        JOIN pg_roles parent ON parent.oid = m.roleid
        WHERE member.rolname IN ('music_owner', 'music_migrator', 'music_api', 'music_monitor')
          AND NOT ((member.rolname = 'music_migrator' AND parent.rolname = 'music_owner')
                OR (member.rolname = 'music_monitor' AND parent.rolname = 'pg_read_all_stats'))
    ) THEN
        RAISE EXCEPTION 'unexpected membership in a PostgreSQL service role';
    END IF;

    IF EXISTS (
        SELECT FROM pg_database d JOIN pg_roles r ON r.oid = d.datdba
        WHERE d.datname = current_database()
          AND r.rolname IN ('music_migrator', 'music_api', 'music_monitor')
    ) THEN
        RAISE EXCEPTION 'a service login owns the music database';
    END IF;
END;
$$;

CREATE FUNCTION pg_temp.set_music_passwords(migrator text, api text, monitor text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF coalesce(migrator, '') = '' OR coalesce(api, '') = '' OR coalesce(monitor, '') = ''
       OR migrator = api OR migrator = monitor OR api = monitor THEN
        RAISE EXCEPTION 'service passwords must be nonempty and distinct';
    END IF;
    EXECUTE format('ALTER ROLE music_migrator PASSWORD %L', migrator);
    EXECUTE format('ALTER ROLE music_api PASSWORD %L', api);
    EXECUTE format('ALTER ROLE music_monitor PASSWORD %L', monitor);
END;
$$;

SELECT pg_temp.set_music_passwords($1, $2, $3)
\bind :migrator_password :api_password :monitor_password
\g

REVOKE ALL ON DATABASE music FROM PUBLIC, music_migrator, music_api, music_monitor;
GRANT CONNECT ON DATABASE music TO music_migrator, music_api, music_monitor;

ALTER SCHEMA public OWNER TO music_owner;
REVOKE ALL ON SCHEMA public FROM PUBLIC, music_migrator, music_api, music_monitor;
GRANT USAGE ON SCHEMA public TO music_api, music_monitor;

SELECT format('ALTER TABLE public.%I OWNER TO music_owner', c.relname)
FROM pg_class c
WHERE c.relnamespace = 'public'::regnamespace AND c.relkind = 'r'
  AND c.relowner <> 'music_owner'::regrole
  AND c.relname IN (
      'account', 'auth_session', 'media_file', 'artist', 'album', 'track',
      'track_artist', 'album_artist', 'album_track', 'playlist', 'playlist_item',
      'favorite_track', 'chart', 'chart_item', 'listening_event', 'schema_migrations'
  )
ORDER BY c.relname
\gexec

SELECT format('ALTER FUNCTION public.%I() OWNER TO music_owner', p.proname)
FROM pg_proc p
WHERE p.pronamespace = 'public'::regnamespace AND p.pronargs = 0 AND p.prokind = 'f'
  AND p.proowner <> 'music_owner'::regrole
  AND p.proname IN ('set_updated_at', 'validate_media_role', 'preserve_media_category')
\gexec

ALTER DEFAULT PRIVILEGES FOR ROLE music_owner REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

ALTER ROLE music_migrator IN DATABASE music SET search_path = public;
ALTER ROLE music_api IN DATABASE music SET search_path = public;
ALTER ROLE music_api IN DATABASE music SET statement_timeout = '5s';
ALTER ROLE music_api IN DATABASE music SET lock_timeout = '2s';
ALTER ROLE music_api IN DATABASE music SET idle_in_transaction_session_timeout = '15s';
ALTER ROLE music_monitor IN DATABASE music SET search_path = pg_catalog;
ALTER ROLE music_monitor IN DATABASE music SET statement_timeout = '5s';
ALTER ROLE music_monitor IN DATABASE music SET default_transaction_read_only = on;

COMMIT;
