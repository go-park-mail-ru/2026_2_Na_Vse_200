BEGIN;

DROP TABLE listening_event;
DROP TABLE chart_item;
DROP TABLE chart;
DROP TABLE favorite_track;
DROP TABLE playlist_item;
DROP TABLE playlist;
DROP TABLE album_track;
DROP TABLE album_artist;
DROP TABLE track_artist;
DROP TABLE track;
DROP TABLE album;
DROP TABLE artist;
DROP TABLE account;
DROP TABLE media_file;

DROP FUNCTION set_updated_at();

COMMIT;
