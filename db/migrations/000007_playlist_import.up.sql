BEGIN;

DELETE FROM playlist_item AS duplicate
USING playlist_item AS first
WHERE duplicate.playlist_id = first.playlist_id
  AND duplicate.track_id = first.track_id
  AND duplicate.position > first.position;

ALTER TABLE playlist_item ADD CONSTRAINT playlist_item_track_uk UNIQUE (playlist_id, track_id);

CREATE TABLE playlist_import (
    owner_id uuid NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    legacy_id text NOT NULL,
    playlist_id uuid UNIQUE REFERENCES playlist (id) ON DELETE SET NULL,
    skipped_track_ids text[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (owner_id, legacy_id)
);

COMMIT;
