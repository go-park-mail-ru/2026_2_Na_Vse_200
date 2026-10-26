# ER Диаграммы модели данных

В PostgreSQL определены 15 прикладных таблиц, включая `auth_session` из миграции `000004_create_auth_session`.
Миграция `000005_media_local_storage` заменяет местоположение медиа на уникальный `storage_key` — относительный путь внутри каталога `MEDIA_ROOT` на диске сервера. Файлы этой миграцией не переносятся. Репозитории БД создают метаданные медиа и связывают аудиозапись с треком; файловый слой для байтов и HTTP-загрузка/скачивание пока не реализованы.

## 1. Основные сущности (PostgreSQL)

```mermaid
classDiagram
direction LR
    class ACCOUNT {
        id [PK]
        email [UK1]
        password_hash
        display_name
        avatar_file_id [FK]
        created_at
        updated_at
    }
    class AUTH_SESSION {
        id [PK]
        account_id [FK]
        expires_at
        created_at
    }
    class MEDIA_FILE {
        id [PK]
        storage_key [UK1]
        mime_type
        byte_size
        duration_ms
        created_at
        updated_at
    }
    class ARTIST {
        id [PK]
        name
        biography
        image_file_id [FK]
        created_at
        updated_at
    }
    class ALBUM {
        id [PK]
        title
        release_date
        cover_file_id [FK]
        created_at
        updated_at
    }
    class TRACK {
        id [PK]
        uploader_id [FK]
        title
        audio_file_id [FK]
        cover_file_id [FK]
        status
        created_at
        updated_at
    }
    ACCOUNT "1" -- "0..*" AUTH_SESSION : has_sessions
    ACCOUNT "1" -- "0..*" TRACK : uploads
    MEDIA_FILE "0..1" -- "0..*" ACCOUNT : avatar_file_id
    MEDIA_FILE "0..1" -- "0..*" ARTIST : image_file_id
    MEDIA_FILE "0..1" -- "0..*" ALBUM : cover_file_id
    MEDIA_FILE "1" -- "0..*" TRACK : audio_file_id
    MEDIA_FILE "0..1" -- "0..*" TRACK : cover_file_id
```

## 2. Исполнители и состав альбомов (PostgreSQL)

```mermaid
classDiagram
direction LR
    class TRACK {
        id [PK]
    }
    class ARTIST {
        id [PK]
    }
    class ALBUM {
        id [PK]
    }
    class TRACK_ARTIST {
        track_id [PK, FK]
        artist_id [PK, FK]
    }
    class ALBUM_ARTIST {
        album_id [PK, FK]
        artist_id [PK, FK]
    }
    class ALBUM_TRACK {
        album_id [PK, FK, UK1]
        track_id [PK, FK]
        track_number [UK1]
    }
    TRACK "1" -- "0..*" TRACK_ARTIST : has
    ARTIST "1" -- "0..*" TRACK_ARTIST : performs
    ALBUM "1" -- "0..*" ALBUM_ARTIST : has
    ARTIST "1" -- "0..*" ALBUM_ARTIST : performs
    ALBUM "1" -- "0..*" ALBUM_TRACK : contains
    TRACK "1" -- "0..*" ALBUM_TRACK : included_in
```

## 3. Плейлисты, чарты и история (PostgreSQL)

```mermaid
classDiagram
direction LR
    class ACCOUNT {
        id [PK]
    }
    class TRACK {
        id [PK]
    }
    class MEDIA_FILE {
        id [PK]
    }
    class PLAYLIST {
        id [PK]
        owner_id [FK]
        title
        description
        cover_file_id [FK]
        is_public
        revision
        created_at
        updated_at
    }
    class PLAYLIST_ITEM {
        id [PK]
        playlist_id [FK, UK1]
        track_id [FK]
        position [UK1]
    }
    class FAVORITE_TRACK {
        account_id [PK, FK]
        track_id [PK, FK]
    }
    class CHART {
        id [PK]
        name
        period_start
        period_end
        cover_file_id [FK]
        created_at
        updated_at
    }
    class CHART_ITEM {
        chart_id [PK, FK, UK1]
        track_id [PK, FK]
        position [UK1]
        streams_count
    }
    class LISTENING_EVENT {
        id [PK]
        account_id [FK]
        track_id [FK]
        started_at
    }
    ACCOUNT "1" -- "0..*" PLAYLIST : owns
    MEDIA_FILE "0..1" -- "0..*" PLAYLIST : cover_file_id
    MEDIA_FILE "0..1" -- "0..*" CHART : cover_file_id
    PLAYLIST "1" -- "0..*" PLAYLIST_ITEM : contains
    TRACK "1" -- "0..*" PLAYLIST_ITEM : added_to
    ACCOUNT "1" -- "0..*" FAVORITE_TRACK : likes
    TRACK "1" -- "0..*" FAVORITE_TRACK : liked_by
    CHART "1" -- "0..*" CHART_ITEM : ranks
    TRACK "1" -- "0..*" CHART_ITEM : ranked_in
    ACCOUNT "1" -- "0..*" LISTENING_EVENT : listens
    TRACK "1" -- "0..*" LISTENING_EVENT : listened_in
```

## 4. Диск сервера и проектируемые Redis и подключения плеера: логические связи с PostgreSQL

`AUTH_SESSION` уже хранится в PostgreSQL. Для файлов выбран диск сервера: `LOCAL_MEDIA_FILE.storage_key` соответствует пути внутри `MEDIA_ROOT`, а содержимое не входит в БД. Метаданные файлов создаются на уровне PostgreSQL-репозиториев; файловый слой, Redis для состояния плеера и очереди и WebSocket пока не реализованы. Диаграмма показывает связи модели этих подсистем. Перенос сессий в Redis не требуется.

```mermaid
classDiagram
direction LR
    class ACCOUNT {
        id [PK]
    }
    class TRACK {
        id [PK]
    }
    class MEDIA_FILE {
        id [PK]
        storage_key [UK1]
    }
    class AUTH_SESSION {
        id [PK]
        account_id [FK]
        expires_at
        created_at
    }
    class PLAYBACK_STATE {
        account_id [PK]
        active_device_id
        current_item_id
        playback_id
        position_ms
        is_playing
        volume
        revision
        updated_at
    }
    class PLAYBACK_QUEUE_ITEM {
        account_id [PK, UK1]
        item_id [PK]
        track_id
        position [UK1]
    }
    class PLAYER_CONNECTION {
        device_id [PK]
        session_id
        connection
    }
    class LOCAL_MEDIA_FILE {
        storage_key [PK]
        binary_content
    }
    class LISTENING_EVENT {
        id [PK]
    }
    ACCOUNT "1" -- "0..*" AUTH_SESSION : has_sessions
    ACCOUNT "1" .. "0..1" PLAYBACK_STATE : syncs_to
    ACCOUNT "1" .. "0..*" PLAYBACK_QUEUE_ITEM : queues
    TRACK "1" .. "0..*" PLAYBACK_QUEUE_ITEM : queued_in
    PLAYBACK_QUEUE_ITEM "0..1" .. "0..1" PLAYBACK_STATE : account_id_current_item_id
    AUTH_SESSION "1" .. "0..*" PLAYER_CONNECTION : session_id
    PLAYER_CONNECTION "0..1" .. "0..1" PLAYBACK_STATE : active_device_id
    LISTENING_EVENT "0..1" .. "0..1" PLAYBACK_STATE : playback_id
    MEDIA_FILE "0..1" .. "1" LOCAL_MEDIA_FILE : storage_key_within_MEDIA_ROOT
```
