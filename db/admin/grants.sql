\set ON_ERROR_STOP on

BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL ROLE music_owner;

REVOKE ALL ON TABLE public.account, public.auth_session, public.media_file, public.artist,
    public.album, public.track, public.track_artist, public.album_artist, public.album_track,
    public.playlist, public.playlist_item, public.favorite_track, public.chart, public.chart_item,
    public.listening_event, public.schema_migrations FROM PUBLIC, music_api, music_monitor;

REVOKE ALL ON FUNCTION public.set_updated_at(), public.validate_media_role(),
    public.preserve_media_category() FROM PUBLIC, music_api, music_monitor;

GRANT SELECT ON TABLE public.account, public.auth_session, public.media_file, public.artist,
    public.album, public.track, public.track_artist, public.album_artist, public.album_track,
    public.favorite_track TO music_api;
GRANT INSERT, DELETE ON TABLE public.account, public.auth_session, public.favorite_track TO music_api;
GRANT INSERT ON TABLE public.media_file TO music_api;
GRANT UPDATE (email, password_hash, display_name, avatar_file_id) ON TABLE public.account TO music_api;

-- validate_media_role() использует FOR KEY SHARE и требует UPDATE хотя бы одного столбца.
GRANT UPDATE (updated_at) ON TABLE public.media_file TO music_api;

-- При удалении аккаунта треки переходят служебному аккаунту, история удаляется.
GRANT UPDATE (uploader_id) ON TABLE public.track TO music_api;
GRANT SELECT (account_id), DELETE ON TABLE public.listening_event TO music_api;

COMMIT;
