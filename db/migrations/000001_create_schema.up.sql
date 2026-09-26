BEGIN;

CREATE TABLE media_file (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    bucket text NOT NULL,
    object_key text NOT NULL,
    mime_type text NOT NULL,
    byte_size bigint NOT NULL,
    duration_ms bigint,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT media_file_pk PRIMARY KEY (id),
    CONSTRAINT media_file_object_key_uk UNIQUE (bucket, object_key),
    CONSTRAINT media_file_bucket_check CHECK (btrim(bucket) <> ''),
    CONSTRAINT media_file_object_key_check CHECK (btrim(object_key) <> ''),
    CONSTRAINT media_file_byte_size_check CHECK (byte_size > 0),
    CONSTRAINT media_file_type_duration_check CHECK (
        (mime_type LIKE 'audio/%' AND duration_ms IS NOT NULL AND duration_ms > 0)
        OR (mime_type LIKE 'image/%' AND duration_ms IS NULL)
    ),
    CONSTRAINT media_file_timestamps_check CHECK (updated_at >= created_at)
);

CREATE TABLE account (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    email text NOT NULL,
    password_hash text NOT NULL,
    display_name text NOT NULL,
    avatar_file_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT account_pk PRIMARY KEY (id),
    CONSTRAINT account_email_uk UNIQUE (email),
    CONSTRAINT account_email_check CHECK (email <> '' AND email = lower(btrim(email))),
    CONSTRAINT account_password_hash_check CHECK (btrim(password_hash) <> ''),
    CONSTRAINT account_display_name_check CHECK (btrim(display_name) <> ''),
    CONSTRAINT account_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT account_avatar_file_fk FOREIGN KEY (avatar_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE artist (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    biography text NOT NULL DEFAULT '',
    image_file_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT artist_pk PRIMARY KEY (id),
    CONSTRAINT artist_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT artist_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT artist_image_file_fk FOREIGN KEY (image_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE album (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    title text NOT NULL,
    release_date date,
    cover_file_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT album_pk PRIMARY KEY (id),
    CONSTRAINT album_title_check CHECK (btrim(title) <> ''),
    CONSTRAINT album_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT album_cover_file_fk FOREIGN KEY (cover_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE track (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    uploader_id uuid NOT NULL,
    title text NOT NULL,
    audio_file_id uuid NOT NULL,
    cover_file_id uuid,
    status text NOT NULL DEFAULT 'draft',
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT track_pk PRIMARY KEY (id),
    CONSTRAINT track_title_check CHECK (btrim(title) <> ''),
    CONSTRAINT track_status_check CHECK (status IN ('draft', 'published', 'unpublished')),
    CONSTRAINT track_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT track_uploader_fk FOREIGN KEY (uploader_id) REFERENCES account (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT track_audio_file_fk FOREIGN KEY (audio_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT track_cover_file_fk FOREIGN KEY (cover_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE track_artist (
    track_id uuid NOT NULL,
    artist_id uuid NOT NULL,
    CONSTRAINT track_artist_pk PRIMARY KEY (track_id, artist_id),
    CONSTRAINT track_artist_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT track_artist_artist_fk FOREIGN KEY (artist_id) REFERENCES artist (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE album_artist (
    album_id uuid NOT NULL,
    artist_id uuid NOT NULL,
    CONSTRAINT album_artist_pk PRIMARY KEY (album_id, artist_id),
    CONSTRAINT album_artist_album_fk FOREIGN KEY (album_id) REFERENCES album (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT album_artist_artist_fk FOREIGN KEY (artist_id) REFERENCES artist (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE album_track (
    album_id uuid NOT NULL,
    track_id uuid NOT NULL,
    track_number integer NOT NULL,
    CONSTRAINT album_track_pk PRIMARY KEY (album_id, track_id),
    CONSTRAINT album_track_number_uk UNIQUE (album_id, track_number) DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT album_track_number_check CHECK (track_number > 0),
    CONSTRAINT album_track_album_fk FOREIGN KEY (album_id) REFERENCES album (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT album_track_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE playlist (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    cover_file_id uuid,
    is_public boolean NOT NULL DEFAULT FALSE,
    revision bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT playlist_pk PRIMARY KEY (id),
    CONSTRAINT playlist_title_check CHECK (btrim(title) <> ''),
    CONSTRAINT playlist_revision_check CHECK (revision >= 0),
    CONSTRAINT playlist_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT playlist_owner_fk FOREIGN KEY (owner_id) REFERENCES account (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT playlist_cover_file_fk FOREIGN KEY (cover_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE playlist_item (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    playlist_id uuid NOT NULL,
    track_id uuid NOT NULL,
    position integer NOT NULL,
    CONSTRAINT playlist_item_pk PRIMARY KEY (id),
    CONSTRAINT playlist_item_position_uk UNIQUE (playlist_id, position) DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT playlist_item_position_check CHECK (position > 0),
    CONSTRAINT playlist_item_playlist_fk FOREIGN KEY (playlist_id) REFERENCES playlist (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT playlist_item_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE favorite_track (
    account_id uuid NOT NULL,
    track_id uuid NOT NULL,
    CONSTRAINT favorite_track_pk PRIMARY KEY (account_id, track_id),
    CONSTRAINT favorite_track_account_fk FOREIGN KEY (account_id) REFERENCES account (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT favorite_track_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE chart (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    cover_file_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chart_pk PRIMARY KEY (id),
    CONSTRAINT chart_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT chart_period_check CHECK (period_end > period_start),
    CONSTRAINT chart_timestamps_check CHECK (updated_at >= created_at),
    CONSTRAINT chart_cover_file_fk FOREIGN KEY (cover_file_id) REFERENCES media_file (id) ON UPDATE RESTRICT ON DELETE SET NULL
);

CREATE TABLE chart_item (
    chart_id uuid NOT NULL,
    track_id uuid NOT NULL,
    position integer NOT NULL,
    streams_count bigint NOT NULL,
    CONSTRAINT chart_item_pk PRIMARY KEY (chart_id, track_id),
    CONSTRAINT chart_item_position_uk UNIQUE (chart_id, position) DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT chart_item_position_check CHECK (position > 0),
    CONSTRAINT chart_item_streams_count_check CHECK (streams_count >= 0),
    CONSTRAINT chart_item_chart_fk FOREIGN KEY (chart_id) REFERENCES chart (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT chart_item_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE listening_event (
    id uuid NOT NULL,
    account_id uuid NOT NULL,
    track_id uuid NOT NULL,
    started_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT listening_event_pk PRIMARY KEY (id),
    CONSTRAINT listening_event_account_fk FOREIGN KEY (account_id) REFERENCES account (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT listening_event_track_fk FOREIGN KEY (track_id) REFERENCES track (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := clock_timestamp();
    RETURN NEW;
END;
$$;

CREATE TRIGGER media_file_updated_at
BEFORE UPDATE ON media_file
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER account_updated_at
BEFORE UPDATE ON account
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER artist_updated_at
BEFORE UPDATE ON artist
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER album_updated_at
BEFORE UPDATE ON album
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER track_updated_at
BEFORE UPDATE ON track
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER playlist_updated_at
BEFORE UPDATE ON playlist
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER chart_updated_at
BEFORE UPDATE ON chart
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX track_uploader_idx ON track (uploader_id);
CREATE INDEX track_audio_file_idx ON track (audio_file_id);
CREATE INDEX track_artist_artist_idx ON track_artist (artist_id);
CREATE INDEX album_artist_artist_idx ON album_artist (artist_id);
CREATE INDEX album_track_track_idx ON album_track (track_id);
CREATE INDEX playlist_owner_idx ON playlist (owner_id);
CREATE INDEX playlist_item_track_idx ON playlist_item (track_id);
CREATE INDEX favorite_track_track_idx ON favorite_track (track_id);
CREATE INDEX chart_item_track_idx ON chart_item (track_id);
CREATE INDEX listening_event_account_started_idx ON listening_event (account_id, started_at DESC, id);
CREATE INDEX listening_event_track_started_idx ON listening_event (track_id, started_at);

COMMIT;
