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

## 4. История прослушиваний и синхронизация плеера

В `listening_event` записываем, кто, какой трек и когда начал слушать. Пауза и переход на другое устройство не считаются новым прослушиванием. Для защиты от повторных записей используем `playback_id` как `id` события. Нужно уточнить у ментора, подходит ли такая история для ДЗ.

Через WebSocket синхронизируем плеер между устройствами пользователя: текущий трек, паузу и перемотку. Планируем воспроизведение на одном устройстве, с остальных можно управлять плеером.

Данные вне PostgreSQL:

- В Redis храним сессии (`session`), состояние плеера (`playback_state`) и очередь треков (`playback_queue_item`).
- В памяти backend храним открытые подключения устройств (`player_connection`).
- В S3 храним аудио и изображения. В `media_file` указываем их расположение: `bucket` и `object_key`.

Связи между этими данными показаны в `er.md`; их проверяет backend.
