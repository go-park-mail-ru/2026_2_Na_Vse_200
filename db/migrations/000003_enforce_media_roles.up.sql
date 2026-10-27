BEGIN;

-- Категория файла сохраняется: для смены audio/image нужна новая запись media_file.
CREATE FUNCTION preserve_media_category()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF split_part(NEW.mime_type, '/', 1) IS DISTINCT FROM split_part(OLD.mime_type, '/', 1) THEN
        RAISE EXCEPTION 'Cannot change media file category from % to %', OLD.mime_type, NEW.mime_type
            USING ERRCODE = '23514', TABLE = TG_TABLE_NAME, COLUMN = 'mime_type';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER media_file_preserve_category
BEFORE UPDATE OF mime_type ON media_file
FOR EACH ROW EXECUTE FUNCTION preserve_media_category();

-- TG_ARGV: имя поля с UUID файла и ожидаемая категория (audio/image).
CREATE FUNCTION validate_media_role()
RETURNS trigger
LANGUAGE plpgsql
SET search_path FROM CURRENT
AS $$
DECLARE
    file_id uuid;
    file_mime text;
BEGIN
    file_id := (to_jsonb(NEW) ->> TG_ARGV[0])::uuid;
    IF file_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT mime_type INTO file_mime
    FROM media_file
    WHERE id = file_id
    FOR KEY SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Media file % does not exist', file_id
            USING ERRCODE = '23503', TABLE = TG_TABLE_NAME, COLUMN = TG_ARGV[0];
    END IF;
    IF file_mime NOT LIKE (TG_ARGV[1] || '/%') THEN
        RAISE EXCEPTION 'Media file % must have category %, got %', file_id, TG_ARGV[1], file_mime
            USING ERRCODE = '23514', TABLE = TG_TABLE_NAME, COLUMN = TG_ARGV[0];
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER account_avatar_media_role
BEFORE INSERT OR UPDATE OF avatar_file_id ON account
FOR EACH ROW EXECUTE FUNCTION validate_media_role('avatar_file_id', 'image');

CREATE TRIGGER artist_image_media_role
BEFORE INSERT OR UPDATE OF image_file_id ON artist
FOR EACH ROW EXECUTE FUNCTION validate_media_role('image_file_id', 'image');

CREATE TRIGGER album_cover_media_role
BEFORE INSERT OR UPDATE OF cover_file_id ON album
FOR EACH ROW EXECUTE FUNCTION validate_media_role('cover_file_id', 'image');

CREATE TRIGGER track_audio_media_role
BEFORE INSERT OR UPDATE OF audio_file_id ON track
FOR EACH ROW EXECUTE FUNCTION validate_media_role('audio_file_id', 'audio');

CREATE TRIGGER track_cover_media_role
BEFORE INSERT OR UPDATE OF cover_file_id ON track
FOR EACH ROW EXECUTE FUNCTION validate_media_role('cover_file_id', 'image');

CREATE TRIGGER playlist_cover_media_role
BEFORE INSERT OR UPDATE OF cover_file_id ON playlist
FOR EACH ROW EXECUTE FUNCTION validate_media_role('cover_file_id', 'image');

CREATE TRIGGER chart_cover_media_role
BEFORE INSERT OR UPDATE OF cover_file_id ON chart
FOR EACH ROW EXECUTE FUNCTION validate_media_role('cover_file_id', 'image');

-- DDL-блокировки удерживаются до COMMIT, поэтому проверка не пропускает
-- конкурентные вставки между проверкой старых данных и включением триггеров.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM (
            SELECT avatar_file_id AS file_id, 'image'::text AS category FROM account
            UNION ALL SELECT image_file_id, 'image' FROM artist
            UNION ALL SELECT cover_file_id, 'image' FROM album
            UNION ALL SELECT audio_file_id, 'audio' FROM track
            UNION ALL SELECT cover_file_id, 'image' FROM track
            UNION ALL SELECT cover_file_id, 'image' FROM playlist
            UNION ALL SELECT cover_file_id, 'image' FROM chart
        ) AS refs
        JOIN media_file AS f ON f.id = refs.file_id
        WHERE f.mime_type NOT LIKE (refs.category || '/%')
    ) THEN
        RAISE EXCEPTION 'Existing media references have incorrect audio/image categories'
            USING ERRCODE = '23514';
    END IF;
END;
$$;

COMMIT;