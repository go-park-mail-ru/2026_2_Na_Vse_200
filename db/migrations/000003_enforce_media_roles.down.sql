BEGIN;

DROP TRIGGER account_avatar_media_role ON account;
DROP TRIGGER artist_image_media_role ON artist;
DROP TRIGGER album_cover_media_role ON album;
DROP TRIGGER track_audio_media_role ON track;
DROP TRIGGER track_cover_media_role ON track;
DROP TRIGGER playlist_cover_media_role ON playlist;
DROP TRIGGER chart_cover_media_role ON chart;
DROP TRIGGER media_file_preserve_category ON media_file;

DROP FUNCTION validate_media_role();
DROP FUNCTION preserve_media_category();

COMMIT;