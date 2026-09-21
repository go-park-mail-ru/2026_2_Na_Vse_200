# ER Диаграммы модели данных

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
    class MEDIA_FILE {
        id [PK]
        bucket [UK1]
        object_key [UK1]
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
        audio_file_id [FK, NOT NULL]
        cover_file_id [FK]
        status
        created_at
        updated_at
    }
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

## 4. Redis, S3 и подключения плеера: логические связи с PostgreSQL

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
        bucket [UK1]
        object_key [UK1]
    }
    class SESSION {
        session_id [PK]
        account_id
        expires_at
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
    class S3_OBJECT {
        bucket [PK]
        object_key [PK]
        binary_content
    }
    class LISTENING_EVENT {
        id [PK]
    }
    ACCOUNT "1" .. "0..*" SESSION : has_active
    ACCOUNT "1" .. "0..1" PLAYBACK_STATE : syncs_to
    ACCOUNT "1" .. "0..*" PLAYBACK_QUEUE_ITEM : queues
    TRACK "1" .. "0..*" PLAYBACK_QUEUE_ITEM : queued_in
    PLAYBACK_QUEUE_ITEM "0..1" .. "0..1" PLAYBACK_STATE : account_id_current_item_id
    SESSION "1" .. "0..*" PLAYER_CONNECTION : session_id
    PLAYER_CONNECTION "0..1" .. "0..1" PLAYBACK_STATE : active_device_id
    LISTENING_EVENT "0..1" .. "0..1" PLAYBACK_STATE : playback_id
    MEDIA_FILE "0..1" .. "1" S3_OBJECT : bucket_object_key
```
