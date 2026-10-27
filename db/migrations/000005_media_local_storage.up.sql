BEGIN;

-- Проверка и преобразование выполняются под одной блокировкой: новые строки
-- не могут появиться между проверкой старого формата и включением ограничений.
LOCK TABLE media_file IN ACCESS EXCLUSIVE MODE;

DO $$
DECLARE
    incompatible_id uuid;
BEGIN
    SELECT id INTO incompatible_id
    FROM media_file
    WHERE strpos(bucket, '/') <> 0
       OR NOT (
           btrim(bucket || '/' || object_key) <> ''
           AND (bucket || '/' || object_key) !~ '(^/|/$|//)'
           AND strpos(bucket || '/' || object_key, chr(92)) = 0
           AND strpos(bucket || '/' || object_key, ':') = 0
           AND (bucket || '/' || object_key) !~ '(^|/)[.]{1,2}(/|$)'
           AND (bucket || '/' || object_key) !~ '[[:cntrl:]]'
       )
    LIMIT 1;

    IF FOUND THEN
        RAISE EXCEPTION 'Cannot migrate media_file % to local storage: incompatible bucket/object_key', incompatible_id
            USING ERRCODE = '23514', TABLE = 'media_file', COLUMN = 'storage_key',
                  DETAIL = 'The legacy bucket must be one path component. bucket/object_key must form a canonical relative path: no absolute path, colon, backslash, empty/dot components, trailing slash or control characters.',
                  HINT = 'Review the original row and plan its file location explicitly; the migration does not rename paths or move files.';
    END IF;
END;
$$;

ALTER TABLE media_file ADD COLUMN storage_key text;

-- Меняется представление местоположения, а не метаданные самого файла.
ALTER TABLE media_file DISABLE TRIGGER media_file_updated_at;
UPDATE media_file SET storage_key = bucket || '/' || object_key;
ALTER TABLE media_file ENABLE TRIGGER media_file_updated_at;

ALTER TABLE media_file
    ALTER COLUMN storage_key SET NOT NULL,
    ADD CONSTRAINT media_file_storage_key_uk UNIQUE (storage_key),
    ADD CONSTRAINT media_file_storage_key_check CHECK (
        btrim(storage_key) <> ''
        AND storage_key !~ '(^/|/$|//)'
        AND strpos(storage_key, chr(92)) = 0
        AND strpos(storage_key, ':') = 0
        AND storage_key !~ '(^|/)[.]{1,2}(/|$)'
        AND storage_key !~ '[[:cntrl:]]'
    ),
    DROP CONSTRAINT media_file_object_key_uk,
    DROP CONSTRAINT media_file_bucket_check,
    DROP CONSTRAINT media_file_object_key_check,
    DROP COLUMN bucket,
    DROP COLUMN object_key;

COMMIT;
