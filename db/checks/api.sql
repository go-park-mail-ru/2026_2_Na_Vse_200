BEGIN;

DO $$
DECLARE
    user_id uuid;
    avatar_id uuid;
    session_id uuid := gen_random_uuid();
    probe_id uuid := gen_random_uuid();
    saved_name text;
BEGIN
    IF current_user <> 'music_api' OR session_user <> 'music_api' THEN
        RAISE EXCEPTION 'connect as music_api';
    END IF;

    INSERT INTO public.account (email, password_hash, display_name)
    VALUES (probe_id || '@example.invalid', 'fixture-hash', 'DB check')
    RETURNING id INTO user_id;

    INSERT INTO public.auth_session (id, account_id, expires_at)
    VALUES (session_id, user_id, now() + interval '1 hour');

    INSERT INTO public.media_file (storage_key, mime_type, byte_size)
    VALUES ('checks/' || probe_id || '.png', 'image/png', 1)
    RETURNING id INTO avatar_id;

    UPDATE public.account SET avatar_file_id = avatar_id, display_name = 'DB check updated'
    WHERE id = user_id RETURNING display_name INTO saved_name;
    IF saved_name IS DISTINCT FROM 'DB check updated' THEN
        RAISE EXCEPTION 'profile update did not return the expected row';
    END IF;

    PERFORM 1 FROM public.track WHERE status = 'published' LIMIT 1;
    UPDATE public.track SET uploader_id = '10000000-0000-4000-8000-000000000001'
    WHERE uploader_id = user_id;
    DELETE FROM public.listening_event WHERE account_id = user_id;
    DELETE FROM public.account WHERE id = user_id;
    IF EXISTS (SELECT FROM public.auth_session WHERE id = session_id) THEN
        RAISE EXCEPTION 'account deletion did not clear the session';
    END IF;

    IF has_table_privilege('public.account', 'TRUNCATE')
       OR has_column_privilege('public.track', 'title', 'UPDATE')
       OR has_column_privilege('public.media_file', 'mime_type', 'UPDATE')
       OR pg_has_role('music_owner', 'SET') THEN
        RAISE EXCEPTION 'music_api has extra privileges';
    END IF;

    BEGIN
        CREATE TABLE public.admin_probe (id int);
        RAISE EXCEPTION 'music_api can create tables';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'CREATE TABLE denied (%)', SQLSTATE;
    END;

    BEGIN
        PERFORM version FROM public.schema_migrations;
        RAISE EXCEPTION 'music_api can read schema_migrations';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'schema_migrations read denied (%)', SQLSTATE;
    END;

    BEGIN
        INSERT INTO public.account (email, password_hash, display_name)
        VALUES ('', 'fixture-hash', 'DB check');
        RAISE EXCEPTION 'account email constraint was not enforced';
    EXCEPTION WHEN check_violation THEN
        RAISE NOTICE 'invalid email rejected (%)', SQLSTATE;
    END;
END;
$$;

ROLLBACK;
