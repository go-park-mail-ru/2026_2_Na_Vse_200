# Модель данных музыкального сервиса

## 1. Описание отношений (PostgreSQL)

- `account`: аккаунт, email, хеш пароля, имя и аватар. Email обязателен и уникален.
- `media_file`: метаданные аудио или изображения в S3. Уникальна пара `(bucket, object_key)`.
- `artist`: исполнитель или группа, имя, биография и изображение.
- `album`: музыкальный релиз, название, дата выпуска и обложка.
- `track`: загруженный пользователем трек, название, аудио, обложка и статус публикации.
- `track_artist`: связь треков и исполнителей.
- `album_artist`: связь альбомов и исполнителей.
- `album_track`: треки альбома и их номера. Треки и номера внутри альбома не повторяются.
- `playlist`: плейлист, владелец, название, описание, обложка, публичность и ревизия.
- `playlist_item`: трек и его позиция в плейлисте. Повторы трека разрешены, позиции уникальны.
- `favorite_track`: избранные треки аккаунта без повторов.
- `chart`: чарт, название, период и обложка.
- `chart_item`: трек, уникальная позиция в чарте и зафиксированное число прослушиваний за период. Число прослушиваний может совпадать у разных треков.
- `listening_event`: запуск трека пользователем и время начала.

Ссылки на изображения необязательны; один файл может использоваться несколькими записями, в том числе аккаунтами. Аудиофайл обязателен для опубликованного трека, но может отсутствовать при загрузке.

В [схемах](er.md): `PK` - первичный ключ, `FK` - внешний ключ, поля с `UK1` внутри сущности вместе образуют альтернативный уникальный ключ. Для `media_file` это одно ограничение `UNIQUE (bucket, object_key)`; оба поля обязательны.

## 2. Функциональные зависимости (ФЗ)

Ниже базовые ФЗ по принятым правилам. Названия, имена и временные метки не уникальны.

```text
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
Ключ: {track_id, artist_id}. Других атрибутов нет.

album_artist:
Ключ: {album_id, artist_id}. Других атрибутов нет.

album_track:
{album_id, track_id} -> track_number
{album_id, track_number} -> track_id

playlist:
{id} -> owner_id, title, description, cover_file_id, is_public, revision, created_at, updated_at

playlist_item:
{id} -> playlist_id, track_id, position
{playlist_id, position} -> id, track_id

favorite_track:
Ключ: {account_id, track_id}. Других атрибутов нет.

chart:
{id} -> name, period_start, period_end, cover_file_id, created_at, updated_at

chart_item:
{chart_id, track_id} -> position, streams_count
{chart_id, position} -> track_id, streams_count

listening_event:
{id} -> account_id, track_id, started_at
```

## 3. Нормальные формы

- **1НФ:** во всех таблицах атомарные атрибуты; списки треков и исполнителей вынесены в таблицы связей.
- **2НФ:** неключевые атрибуты зависят от полного кандидатного ключа. В `media_file`, `album_track`, `playlist_item`, `chart_item` отдельные части составных ключей запись не определяют. В `track_artist`, `album_artist`, `favorite_track` других атрибутов нет; у остальных таблиц ключи одноатрибутные.
- **3НФ:** зависимостей между неключевыми атрибутами по принятым правилам нет. Например, `display_name` зависит и от `id`, и от `email`, но оба являются кандидатными ключами, поэтому нарушения нет.
- **НФБК:** определители всех перечисленных нетривиальных ФЗ - кандидатные ключи: `id`, альтернативные `email` и составные ключи из раздела 2. В таблицах связей без других атрибутов нет нетривиальных ФЗ. Следовательно, все 14 таблиц находятся в НФБК при указанных правилах.

## 4. История, WebSocket и остальные хранилища

**История:** подтверждённый новый запуск создаёт `listening_event`, его `id` совпадает с `playback_id`. Повторная доставка, пауза и смена устройства не создают дублей. События сохраняются при снятии трека с публикации. Применимость истории прослушиваний к требованию историчности нужно согласовать с ментором.

**WebSocket:** синхронизация запуска, паузы, перемотки и смены трека между устройствами аккаунта. Предлагается одно звучащее устройство; остальные отображают состояние и управляют им. Аудио загружается из S3.

**Redis:**

- `session`: {session_id} -> account_id, expires_at; сессия с TTL.
- `playback_state`: {account_id} -> active_device_id, current_item_id, playback_id, position_ms, is_playing, volume, revision, updated_at.
- `playback_queue_item`: {account_id, item_id} -> track_id, position; {account_id, position} -> item_id, track_id.

**Память backend:** `player_connection`: {device_id} -> session_id, connection; активные подключения плеера.

**S3:** `s3_object`: {bucket, object_key} -> binary_content; аудио и изображения. `media_file` ссылается на объект по этой паре; у загружаемого объекта метаданные могут ещё отсутствовать.

Связи между хранилищами проверяет backend. Текущий элемент очереди определяется парой `(account_id, current_item_id)`, активное устройство принадлежит тому же аккаунту, `playback_id` связывает запуск с историей.
