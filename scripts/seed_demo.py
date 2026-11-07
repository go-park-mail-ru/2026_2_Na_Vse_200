"""Create a repeatable local demo catalog and generated WAV files (stdlib only)."""

import math
from pathlib import Path
import struct
import subprocess
import uuid
import wave


ROOT = Path(__file__).resolve().parents[1]
NAMESPACE = uuid.UUID("3c1b4bc9-0191-4cc7-9210-ef538cd85747")
UPLOADER = "10000000-0000-4000-8000-000000000001"


def identifier(key):
    return str(uuid.uuid5(NAMESPACE, key))


def literal(value):
    return "'" + str(value).replace("'", "''") + "'"


def main():
    artist = identifier("artist")
    album = identifier("album")
    statements = [
        "BEGIN;",
        "INSERT INTO artist (id, name, biography) VALUES "
        f"('{artist}', 'На все 200 — демо', "
        "'Тестовый исполнитель для проверки страницы. Аудио — сгенерированные звуковые примеры.') "
        "ON CONFLICT (id) DO NOTHING;",
        "INSERT INTO album (id, title, release_date) VALUES "
        f"('{album}', 'Первый запуск', '2026-11-07') ON CONFLICT (id) DO NOTHING;",
        f"INSERT INTO album_artist (album_id, artist_id) VALUES ('{album}', '{artist}') "
        "ON CONFLICT DO NOTHING;",
    ]
    for number, (title, frequency) in enumerate([
        ("Утро — демо", 220), ("Дорога — демо", 330), ("Вечер — демо", 440),
    ], start=1):
        track = identifier(f"track-{number}")
        media = identifier(f"audio-{number}")
        storage_key = f"audio/demo/{track}.wav"
        path = ROOT / "media" / storage_key
        path.parent.mkdir(parents=True, exist_ok=True)
        if not path.exists():
            rate, seconds = 8000, 8
            frames = bytearray()
            for sample in range(rate * seconds):
                time = sample / rate
                fade = min(time / 0.1, (seconds - time) / 0.1, 1)
                amplitude = int(4000 * fade * math.sin(2 * math.pi * frequency * time))
                frames.extend(struct.pack("<h", amplitude))
            with wave.open(str(path), "wb") as audio:
                audio.setparams((1, 2, rate, 0, "NONE", "not compressed"))
                audio.writeframes(frames)
        with wave.open(str(path), "rb") as audio:
            duration = round(audio.getnframes() * 1000 / audio.getframerate())
        statements.extend([
            "INSERT INTO media_file (id, storage_key, mime_type, byte_size, duration_ms) VALUES "
            f"('{media}', '{storage_key}', 'audio/wav', {path.stat().st_size}, {duration}) "
            "ON CONFLICT (id) DO NOTHING;",
            "INSERT INTO track (id, uploader_id, title, audio_file_id, status) VALUES "
            f"('{track}', '{UPLOADER}', {literal(title)}, '{media}', 'published') "
            "ON CONFLICT (id) DO NOTHING;",
            f"INSERT INTO track_artist (track_id, artist_id) VALUES ('{track}', '{artist}') "
            "ON CONFLICT DO NOTHING;",
            f"INSERT INTO album_track (album_id, track_id, track_number) VALUES ('{album}', '{track}', {number}) "
            "ON CONFLICT (album_id, track_id) DO NOTHING;",
        ])
    statements.append("COMMIT;")
    subprocess.run(
        ["docker", "compose", "exec", "-T", "db", "psql", "-U", "music", "-d", "music", "-v", "ON_ERROR_STOP=1"],
        cwd=ROOT, input="\n".join(statements).encode("utf-8"), check=True,
    )
    print(f"Artist page: http://127.0.0.1:5173/artists/{artist}")


if __name__ == "__main__":
    main()
