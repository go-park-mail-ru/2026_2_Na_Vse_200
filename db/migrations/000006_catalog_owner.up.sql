BEGIN;

ALTER TABLE account ADD COLUMN system_key text UNIQUE;
COMMENT ON COLUMN account.system_key IS 'Устойчивое назначение служебного аккаунта, независимое от UUID';
UPDATE account SET system_key = 'catalog_owner' WHERE email = 'catalog@navse200.ru';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM account WHERE system_key = 'catalog_owner') THEN
        RAISE EXCEPTION 'Catalog owner account is missing';
    END IF;
END $$;

COMMIT;
