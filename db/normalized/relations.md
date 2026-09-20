## 1. Описание отношений (таблицы PostgreSQL)

- `account`: Аккаунт пользователя. Хранит email, хеш пароля, отображаемое имя и ссылку на аватар
- `media_file`: Метаданные файла в S3. Хранит bucket, object_key, mime_type, byte_size, duration_ms
- `artist`: Исполнитель или группа. Хранит имя, биографию и ссылку на изображение
- `album`: Музыкальный релиз. Хранит название, дату выпуска и ссылку на обложку
- `track`: Аудиозапись. Хранит название, id загрузившего пользователя, ссылки на аудио и обложку, статус публикации
- `track_artist`: Связь многие-ко-многим между треком и исполнителем
- `album_artist`: Связь многие-ко-многим между альбомом и исполнителем
- `album_track`: Состав альбома. Хранит номер трека (позицию) в конкретном альбоме
- `playlist`: Пользовательский плейлист. Хранит владельца, название, описание, публичность и ревизию
- `playlist_item`: Элемент плейлиста. Хранит позицию конкретного трека в конкретном плейлисте
- `favorite_track`: Избранные треки пользователя (связь многие-ко-многим)
- `chart`: чарт. Хранит название, период действия и обложку
- `chart_item`: Элемент чарта. Хранит позицию трека в конкретном чарте и количество стримов за период
- `listening_event`: Событие прослушивания. Хранит факт запуска трека пользователем во времени

## 2. Функциональные зависимости (ФЗ)

account:
{id} -> email, password_hash, display_name, avatar_file_id, created_at, updated_at
{email} -> id, password_hash, display_name, avatar_file_id, created_at, updated_at

media_file:
{id} -> bucket, object_key, mime_type, byte_size, duration_ms, created_at, updated_at
{bucket, object_key} -> id, mime_type, byte_size, duration_ms, created_at, updated_at

artist:
{id} -> name, biography, image_file_id, created_at, updated_at

album:
{id} -> title, release_date, cover_file_id, created_at, updated_at

track:
{id} -> uploader_id, title, audio_file_id, cover_file_id, status, created_at, updated_at

track_artist:
{track_id, artist_id} -> (нет неключевых атрибутов, все атрибуты входят в первичный ключ)

album_artist:
{album_id, artist_id} -> (нет неключевых атрибутов, все атрибуты входят в первичный ключ)

album_track:
{album_id, track_id} -> track_number
{album_id, track_number} -> track_id

playlist:
{id} -> owner_id, title, description, cover_file_id, is_public, revision, created_at, updated_at

playlist_item:
{id} -> playlist_id, track_id, position
{playlist_id, position} -> id, track_id

favorite_track:
{account_id, track_id} -> (нет неключевых атрибутов)

chart:
{id} -> name, period_start, period_end, cover_file_id, created_at, updated_at

chart_item:
{chart_id, track_id} -> position, streams_count
{chart_id, position} -> track_id, streams_count

listening_event:
{id} -> account_id, track_id, started_at

## 3. Доказательство нормальных форм

Для всех вышеописанных отношений выполняется следующее:

1НФ: 
Все атрибуты являются атомарными. Запрещено использование массивов, JSON или вложенных структур. Списки исполнителей или треков вынесены в отдельные отношения связи (`track_artist`, `album_track`, `chart_item`).

2НФ: 
Все неключевые атрибуты полностью функционально зависят от всего первичного ключа, а не от его части. 
- В отношениях с составным ключом (`track_artist`, `album_artist`, `favorite_track`, `chart_item`) нет неключевых атрибутов, которые зависели бы только от части ключа. 
- В `album_track` атрибут `track_number` зависит от полного составного ключа `{album_id, track_id}`.

3НФ: 
Отсутствуют транзитивные зависимости неключевых атрибутов от первичного ключа. 
- Например, `duration_ms` зависит только от `media_file.id`, а не от `track.id`. Мы не дублируем длительность в таблице `track`.
- `display_name` зависит только от `account.id`, а не от других полей аккаунта.

НФБК: 
В каждом отношении каждая нетривиальная функциональная зависимость имеет в качестве детерминанта потенциальный ключ.
- В `account` детерминантами являются `{id}` и `{email}`, оба являются кандидатными ключами (AK).
- В `media_file` детерминантами являются `{id}` и `{bucket, object_key}`, оба являются кандидатными ключами.
- В `playlist_item` и `chart_item` детерминантами являются `{id}` (или составной PK) и альтернативные ключи `{playlist_id, position}` / `{chart_id, position}`, которые также являются кандидатными ключами (обеспечивают уникальность позиции в рамках одного списка).
- В `album_track` детерминантами являются `{album_id, track_id}` и `{album_id, track_number}`, оба являются кандидатными ключами.
- Во всех остальных таблицах первичный ключ является единственным детерминантом.
Следовательно, все отношения находятся в НФБК.

## 4. Историчность и WebSocket

Redis:
- `session`: {session_id} -> account_id, expires_at (TTL)
- `playback_state`: {account_id} -> active_device_id, current_item_id, playback_id, position_ms, is_playing, volume, revision, updated_at
- `playback_queue_item`: {account_id, item_id} -> track_id, position

S3:
- `s3_object`: {bucket, object_key} -> binary_content