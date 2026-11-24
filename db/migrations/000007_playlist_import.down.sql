BEGIN;
DROP TABLE playlist_import;
ALTER TABLE playlist_item DROP CONSTRAINT playlist_item_track_uk;
COMMIT;
