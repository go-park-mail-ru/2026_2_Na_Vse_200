BEGIN;

-- Local demonstration accounts. Passwords and purpose are documented in db/README.md.
-- Each bcrypt hash has its own salt and cost 12; plaintext passwords are not stored.
INSERT INTO account (id, email, password_hash, display_name)
VALUES
    (
        '10000000-0000-4000-8000-000000000001',
        'demo.listener@example.test',
        '$2a$12$6mmSQ5iYb/2VgJuzSOKGIuYvfSQWR/kv2MmafaEU8q.PDQJNFFhVa',
        'Demo Listener'
    ),
    (
        '10000000-0000-4000-8000-000000000002',
        'demo.uploader@example.test',
        '$2a$12$1KzNA7EpdS8UJE/cOCd/DucczaOzjMuYQT/Vrd.VeH8nsDRQX7Mqm',
        'Demo Uploader'
    );

COMMIT;
