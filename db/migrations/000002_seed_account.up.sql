BEGIN;

-- Служебный аккаунт: от его имени в каталоге числятся треки (track.uploader_id).
-- Пароль случайный и нигде не хранится, войти под этим аккаунтом нельзя.
INSERT INTO account (id, email, password_hash, display_name)
VALUES (
    '10000000-0000-4000-8000-000000000001',
    'catalog@navse200.ru',
    '$2a$10$WQNfVR99AY.RUDvS.OXsq.OKF4.p0y3vjATVIP.AxUI9RD7xcrXLm',
    'Каталог «На все 200»'
);

COMMIT;

