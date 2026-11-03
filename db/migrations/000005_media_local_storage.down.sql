BEGIN;

LOCK TABLE media_file IN ACCESS EXCLUSIVE MODE;

-- UP сохраняет границу legacy bucket первым '/'. Для новых однокомпонентных
-- ключей такой границы нет: её нельзя выдумывать или молча менять путь.
DO $$
DECLARE
    incompatible_id uuid;
BEGIN
    SELECT id INTO incompatible_id
    FROM media_file
    WHERE strpos(storage_key, '/') = 0
       OR btrim(split_part(storage_key, '/', 1)) = ''
       OR btrim(substring(storage_key FROM strpos(storage_key, '/') + 1)) = ''
    LIMIT 1;

    IF FOUND THEN
        RAISE EXCEPTION 'Cannot roll back local storage for media_file %: storage_key has no compatible bucket/object_key split', incompatible_id
            USING ERRCODE = '23514', TABLE = 'media_file', COLUMN = 'storage_key',
                  DETAIL = 'Rollback splits storage_key at its first slash. Both parts must satisfy the previous nonblank checks; single-component keys cannot be rolled back automatically.',
                  HINT = 'Keep the current schema or prepare an explicit data-and-file migration. This rollback neither rewrites keys nor moves files.';
    END IF;
END;
$$;

ALTER TABLE media_file
    ADD COLUMN bucket text,
    ADD COLUMN object_key text;

ALTER TABLE media_file DISABLE TRIGGER media_file_updated_at;
UPDATE media_file
SET bucket = split_part(storage_key, '/', 1),
    object_key = substring(storage_key FROM strpos(storage_key, '/') + 1);
ALTER TABLE media_file ENABLE TRIGGER media_file_updated_at;

ALTER TABLE media_file
    ALTER COLUMN bucket SET NOT NULL,
    ALTER COLUMN object_key SET NOT NULL,
    ADD CONSTRAINT media_file_object_key_uk UNIQUE (bucket, object_key),
    ADD CONSTRAINT media_file_bucket_check CHECK (btrim(bucket) <> ''),
    ADD CONSTRAINT media_file_object_key_check CHECK (btrim(object_key) <> ''),
    DROP CONSTRAINT media_file_storage_key_uk,
    DROP CONSTRAINT media_file_storage_key_check,
    DROP COLUMN storage_key;

COMMIT;
