## 1. Основные сущности (PostgreSQL)

```mermaid
erDiagram
    ACCOUNT {
        id PK
        email UK
        password_hash
        display_name
        avatar_file_id FK
        created_at
        updated_at
    }
    MEDIA_FILE {
        id PK
        bucket
        object_key UK
        mime_type
        byte_size
        duration_ms
        created_at
        updated_at
    }
    ARTIST {
        id PK
        name
        biography
        image_file_id FK
        created_at
        updated_at
    }
    ALBUM {
        id PK
        title
        release_date
        cover_file_id FK
        created_at
        updated_at
    }
    TRACK {
        id PK
        uploader_id FK
        title
        audio_file_id FK
        cover_file_id FK
        status
        created_at
        updated_at
    }

    ACCOUNT ||--o{ TRACK : "uploads"
    ACCOUNT ||--o| MEDIA_FILE : "avatar"
    ARTIST ||--o| MEDIA_FILE : "image"
    ALBUM ||--o| MEDIA_FILE : "cover"
    TRACK ||--o| MEDIA_FILE : "audio"
    TRACK ||--o| MEDIA_FILE : "cover"


    erDiagram
    TRACK {
        id PK
    }
    ARTIST {
        id PK
    }
    ALBUM {
        id PK
    }
    TRACK_ARTIST {
        track_id PK,FK
        artist_id PK,FK
    }
    ALBUM_ARTIST {
        album_id PK,FK
        artist_id PK,FK
    }
    ALBUM_TRACK {
        album_id PK,FK
        track_id PK,FK
        track_number
    }

    TRACK ||--o{ TRACK_ARTIST : "has"
    ARTIST ||--o{ TRACK_ARTIST : "performs"
    ALBUM ||--o{ ALBUM_ARTIST : "has"
    ARTIST ||--o{ ALBUM_ARTIST : "performs"
    ALBUM ||--o{ ALBUM_TRACK : "contains"
    TRACK ||--o{ ALBUM_TRACK : "included_in"

    erDiagram
    ACCOUNT {
        id PK
    }
    TRACK {
        id PK
    }
    PLAYLIST {
        id PK
        owner_id FK
        title
        description
        cover_file_id FK
        is_public
        revision
        created_at
        updated_at
    }
    PLAYLIST_ITEM {
        id PK
        playlist_id FK
        track_id FK
        position
    }
    FAVORITE_TRACK {
        account_id PK,FK
        track_id PK,FK
    }
    CHART {
        id PK
        name
        period_start
        period_end
        cover_file_id FK
        created_at
        updated_at
    }
    CHART_ITEM {
        chart_id PK,FK
        track_id PK,FK
        position
        streams_count
    }
    LISTENING_EVENT {
        id PK
        account_id FK
        track_id FK
        started_at
    }

    ACCOUNT ||--o{ PLAYLIST : "owns"
    PLAYLIST ||--o{ PLAYLIST_ITEM : "contains"
    TRACK ||--o{ PLAYLIST_ITEM : "added_to"
    ACCOUNT ||--o{ FAVORITE_TRACK : "likes"
    TRACK ||--o{ FAVORITE_TRACK : "liked_by"
    CHART ||--o{ CHART_ITEM : "ranks"
    TRACK ||--o{ CHART_ITEM : "ranked_in"
    ACCOUNT ||--o{ LISTENING_EVENT : "listens"
    TRACK ||--o{ LISTENING_EVENT : "listened_in"

    erDiagram
    ACCOUNT {
        id PK
    }
    TRACK {
        id PK
    }
    MEDIA_FILE {
        id PK
        bucket FK
        object_key FK
    }
    SESSION {
        session_id PK
        account_id FK
        expires_at
    }
    PLAYBACK_STATE {
        account_id PK
        active_device_id
        current_item_id
        playback_id
        position_ms
        is_playing
        volume
        revision
        updated_at
    }
    PLAYBACK_QUEUE_ITEM {
        account_id PK
        item_id PK
        track_id FK
        position
    }
    S3_OBJECT {
        bucket PK
        object_key PK
        binary_content
    }

    ACCOUNT ||--o{ SESSION : "has_active"
    ACCOUNT ||--o| PLAYBACK_STATE : "syncs_to"
    ACCOUNT ||--o{ PLAYBACK_QUEUE_ITEM : "queues"
    TRACK ||--o{ PLAYBACK_QUEUE_ITEM : "queued_in"
    MEDIA_FILE ||--|| S3_OBJECT : "points_to"