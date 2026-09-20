## 1. Описание отношений (таблицы PostgreSQL)

- `account`: Аккаунт пользователя. Хранит email, хеш пароля, отображаемое имя и необязательную ссылку `avatar_file_id` на `media_file.id`. Первичный ключ — `id`, альтернативный кандидатный ключ — обязательный уникальный `email`. Один файл может не использоваться как аватар либо использоваться несколькими аккаунтами; UNIQUE на `avatar_file_id` не предполагается. Эта связь не означает владение файлом.
- `media_file`: Метаданные файла в S3. Хранит bucket, object_key, mime_type, byte_size, duration_ms. Первичный ключ — `id`, альтернативный кандидатный ключ — обязательная пара `(bucket, object_key)`. Уникальность `object_key` отдельно не требуется: одинаковый ключ допустим в разных bucket. При реализации нужно одно ограничение `UNIQUE (bucket, object_key)`, а не отдельные UNIQUE на каждый атрибут.
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

Ссылки на изображения и обложки необязательны и допускают повторное использование файла. У опубликованного трека аудиофайл обязателен; на стадии загрузки `audio_file_id` может отсутствовать.

В [ER-диаграмме](er.md) пометки `UK` у `media_file.bucket` и `media_file.object_key` обозначают участие в одном составном ключе `(bucket, object_key)`, а не отдельную уникальность каждого поля.

## 2. Функциональные зависимости (ФЗ)

Перечислены базовые нетривиальные зависимости по принятым бизнес-правилам; остальные выводятся из них. Названия и отображаемые имена не уникальны. Время создания/изменения само по себе не определяет другие атрибуты.

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

## 3. Доказательство нормальных форм

Для всех вышеописанных отношений выполняется следующее:

1НФ: 
Все атрибуты являются атомарными. Запрещено использование массивов, JSON или вложенных структур. Списки исполнителей или треков вынесены в отдельные отношения связи (`track_artist`, `album_track`, `chart_item`).

2НФ: 
Все неключевые атрибуты полностью функционально зависят от каждого кандидатного ключа, а не от его части.

- В `track_artist`, `album_artist`, `favorite_track` неключевых атрибутов нет.
- В `album_track` место определяется парой `{album_id, track_id}`, а трек на конкретном месте — парой `{album_id, track_number}`.
- В `chart_item` количество прослушиваний зависит от трека в конкретном чарте; место также уникально только внутри чарта. В `playlist_item` позиция определяет элемент только вместе с `playlist_id`.
- В `media_file` отдельный `bucket` или отдельный `object_key` не определяет файл; нужен полный составной ключ. Для остальных отношений кандидатные ключи одноатрибутные, частичная зависимость невозможна.

3НФ: 
В каждой нетривиальной ФЗ определитель является суперключом либо зависимый атрибут входит в кандидатный ключ. В описанной модели все приведённые нетривиальные ФЗ удовлетворяют первому условию.

- В `account` атрибут `display_name` определяется **и `{id}`, и `{email}`**. Оба определителя — кандидатные ключи: `id` выбран первичным, `email` — альтернативным. Поэтому цепочка `id → email → display_name` не нарушает 3НФ: промежуточный `email` сам является ключом, а не неключевым атрибутом.
- Аналогично `media_file.duration_ms` определяется и `{id}`, и `{bucket, object_key}`. Длительность хранится у файла и не дублируется в `track`.
- Данные аккаунта, исполнителя и файла не копируются в таблицы связей; дополнительных зависимостей между неключевыми атрибутами по принятым правилам нет.

НФБК: 
В каждом отношении определитель любой нетривиальной ФЗ является суперключом. У приведённых базовых ФЗ определители — кандидатные ключи; их надмножества тоже являются суперключами.

- В `account` определители `{id}` и `{email}` — кандидатные ключи; выбранный PK — `{id}`, альтернативный ключ (AK) — `{email}`.
- В `media_file` детерминантами являются `{id}` и `{bucket, object_key}`, оба являются кандидатными ключами.
- В `playlist_item` и `chart_item` детерминантами являются `{id}` (или составной PK) и альтернативные ключи `{playlist_id, position}` / `{chart_id, position}`, которые также являются кандидатными ключами (обеспечивают уникальность позиции в рамках одного списка).
- В `album_track` детерминантами являются `{album_id, track_id}` и `{album_id, track_number}`, оба являются кандидатными ключами.
- В `track_artist`, `album_artist`, `favorite_track` нет нетривиальных ФЗ, поэтому нарушения НФБК нет.
- В `artist`, `album`, `track`, `playlist`, `chart`, `listening_event` определителем базовых нетривиальных ФЗ является `{id}` — единственный кандидатный ключ.

Следовательно, все перечисленные отношения находятся в НФБК в рамках зафиксированных бизнес-правил и функциональных зависимостей. При изменении правил зависимости и ключи нужно проверить повторно.

## 4. Историчность и WebSocket

`session`, `playback_state`, `playback_queue_item` хранятся в Redis, `s3_object` — в S3. Связи этих структур с PostgreSQL проверяет приложение; это не ограничения FOREIGN KEY PostgreSQL.

Redis:
- `session`: {session_id} -> account_id, expires_at (TTL)
- `playback_state`: {account_id} -> active_device_id, current_item_id, playback_id, position_ms, is_playing, volume, revision, updated_at
- `playback_queue_item`: {account_id, item_id} -> track_id, position

S3:
- `s3_object`: {bucket, object_key} -> binary_content

Запись `media_file` указывает на один объект по паре `(bucket, object_key)`. У объекта S3 может ещё не быть строки метаданных, например пока загрузка не завершена. Наличие объекта проверяется при завершении загрузки; случаи утраты файла обрабатывает приложение.
