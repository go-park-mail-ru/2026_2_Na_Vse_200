BEGIN;

INSERT INTO account (id, email, password_hash, display_name)
VALUES
    (
        '10000000-0000-4000-8000-000000000001',
        'acc1@example.com',
        '$2a$12$6mmSQ5iYb/2VgJuzSOKGIuYvfSQWR/kv2MmafaEU8q.PDQJNFFhVa',
        'Аккаунт 1'
    ),
    (
        '10000000-0000-4000-8000-000000000002',
        'acc2@example.com',
        '$2a$12$1KzNA7EpdS8UJE/cOCd/DucczaOzjMuYQT/Vrd.VeH8nsDRQX7Mqm',
        'Аккаунт 2'
    );

COMMIT;
