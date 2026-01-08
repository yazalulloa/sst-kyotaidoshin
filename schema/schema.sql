-- DROP TABLE IF EXISTS rates;
-- DROP TABLE IF EXISTS apartments;
-- DROP TABLE IF EXISTS buildings;
-- DROP TABLE IF EXISTS extra_charges;
-- DROP TABLE IF EXISTS reserve_funds;
-- DROP TABLE IF EXISTS receipts;
-- DROP TABLE IF EXISTS expenses;
-- DROP TABLE IF EXISTS debts;
-- DROP TABLE IF EXISTS users;
-- DROP TABLE IF EXISTS roles;
-- DROP TABLE IF EXISTS permissions;
-- DROP TABLE IF EXISTS role_permissions;
-- DROP TABLE IF EXISTS user_roles;



-- DROP TABLE IF EXISTS rates;

CREATE TABLE IF NOT EXISTS rates
(
    id            BIGINT PRIMARY KEY,
    from_currency TEXT                                             NOT NULL,
    to_currency   TEXT                                             NOT NULL,
    rate          DECIMAL(16, 12)                                  NOT NULL,
    source        TEXT                                             NOT NULL,
    trend         TEXT CHECK ( trend IN ('UP', 'DOWN', 'STABLE') ) NOT NULL,
    diff          DECIMAL(16, 12)                                  NOT NULL,
    diff_percent  DECIMAL(16, 2)                                   NOT NULL,
    date_of_rate  DATE                                             NOT NULL,
    date_of_file  DATE                                             NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ALTER TABLE rates ADD COLUMN trend TEXT NOT NULL DEFAULT 'STABLE';

CREATE INDEX IF NOT EXISTS rates_from_currency_to_currency_rate_date_of_rate_idx
    ON rates (from_currency, to_currency, rate, date_of_rate);

CREATE INDEX IF NOT EXISTS rates_from_currency_idx ON rates (from_currency);
CREATE INDEX IF NOT EXISTS rates_date_of_rate_idx ON rates (date_of_rate);
CREATE INDEX IF NOT EXISTS rates_from_currency_date_of_rate_idx ON rates (from_currency, date_of_rate);
CREATE INDEX IF NOT EXISTS rates_trend_idx ON rates (trend);


-- DROP TABLE IF EXISTS apartments;
CREATE TABLE IF NOT EXISTS apartments
(
    building_id CHAR(20)      NOT NULL,
    number      CHAR(20)      NOT NULL,
    name        VARCHAR(100)  NOT NULL,
    id_doc      CHAR(20)      NOT NULL,
    aliquot     DECIMAL(3, 2) NOT NULL DEFAULT 0.00,
    emails      TEXT          NOT NULL,
    created_at  DATETIME      DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME,
    PRIMARY KEY (building_id, number)
);

CREATE INDEX IF NOT EXISTS apartments_building_id_number_idx ON apartments (building_id, number);
CREATE INDEX IF NOT EXISTS apartments_building_id_number_name_emails_idx ON apartments (building_id, number, name, emails);

CREATE TRIGGER IF NOT EXISTS apartments_updated_at_trigger
    AFTER UPDATE
    ON apartments
    FOR EACH ROW
BEGIN
    UPDATE apartments SET updated_at = CURRENT_TIMESTAMP WHERE building_id = OLD.building_id AND number = OLD.number;
END;

-- ALTER TABLE buildings ADD COLUMN debts_currencies_to_show TEXT NOT NULL DEFAULT '';

-- DROP TABLE IF EXISTS buildings;
CREATE TABLE IF NOT EXISTS buildings
(
    id                               CHAR(20)                                       NOT NULL UNIQUE PRIMARY KEY,
    name                             VARCHAR(100)                                   NOT NULL,
    rif                              CHAR(20)                                       NOT NULL,
    main_currency                    TEXT CHECK ( main_currency IN ('USD', 'VED') ) NOT NULL,
    debt_currency                    TEXT CHECK ( debt_currency IN ('USD', 'VED') ) NOT NULL,
    currencies_to_show_amount_to_pay TEXT                                           NOT NULL,
    debts_currencies_to_show         TEXT                                           NOT NULL,
    fixed_pay                        BOOL                                           NOT NULL,
    fixed_pay_amount                 DECIMAL(16, 2)                                 NOT NULL,
    round_up_payments                BOOL                                           NOT NULL,
    email_config                     TEXT                                           NOT NULL,
    created_at                       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at                       DATETIME
);

CREATE TRIGGER IF NOT EXISTS buildings_updated_at_trigger
    AFTER UPDATE
    ON buildings
    FOR EACH ROW
BEGIN
UPDATE buildings SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

-- DROP TABLE IF EXISTS extra_charges;
CREATE TABLE IF NOT EXISTS extra_charges
(
    id               INTEGER PRIMARY KEY,
    building_id      CHAR(20)                                       NOT NULL,
    parent_reference CHAR(20)                                       NOT NULL,
    type             TEXT CHECK ( type IN ('BUILDING', 'RECEIPT') ) NOT NULL,
    description      VARCHAR(100)                                   NOT NULL,
    amount           DECIMAL(16, 2)                                 NOT NULL,
    currency         TEXT CHECK ( currency IN ('USD', 'VED') )      NOT NULL,
    active           BOOL                                           NOT NULL,
    apartments       TEXT                                           NOT NULL,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME
);


CREATE INDEX IF NOT EXISTS extra_charges_parent_reference_idx ON extra_charges (parent_reference);
CREATE INDEX IF NOT EXISTS extra_charges_building_id_idx ON extra_charges (building_id);
CREATE INDEX IF NOT EXISTS extra_charges_parent_reference_building_id_idx ON extra_charges (parent_reference, building_id);
CREATE INDEX IF NOT EXISTS extra_charges_type_idx ON extra_charges (type);

CREATE INDEX IF NOT EXISTS extra_charges_parent_reference_type_idx ON extra_charges (parent_reference, type);

CREATE TRIGGER IF NOT EXISTS extra_charges_updated_at_trigger
    AFTER UPDATE
                                ON extra_charges
                                FOR EACH ROW
BEGIN
UPDATE extra_charges
SET updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
END;

-- DROP TABLE IF EXISTS reserve_funds;
CREATE TABLE IF NOT EXISTS reserve_funds
(
    id              INTEGER PRIMARY KEY,
    building_id     CHAR(20)                                              NOT NULL,
    name            VARCHAR(100)                                          NOT NULL,
    fund            DECIMAL(16, 2)                                        NOT NULL,
    expense         DECIMAL(16, 2)                                        NOT NULL,
    pay             DECIMAL(16, 2)                                        NOT NULL,
    active          BOOL                                                  NOT NULL,
    type            TEXT CHECK ( type IN ('FIXED_PAY', 'PERCENTAGE') )    NOT NULL,
    expense_type    TEXT CHECK ( expense_type IN ('COMMON', 'UNCOMMON') ) NOT NULL,
    add_to_expenses BOOL                                                  NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME
    );

CREATE INDEX IF NOT EXISTS reserve_funds_building_id_idx ON reserve_funds (building_id);

CREATE TRIGGER IF NOT EXISTS reserve_funds_updated_at_trigger
    AFTER UPDATE
                                ON reserve_funds
                                FOR EACH ROW
BEGIN
UPDATE reserve_funds
SET updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
END;

-- DROP TABLE IF EXISTS receipts;
CREATE TABLE IF NOT EXISTS receipts
(
    id          VARCHAR(50)         NOT NULL,
    building_id CHAR(20)            NOT NULL,
    year        SMALLINT            NOT NULL,
    month       SMALLINT            NOT NULL,
    date        DATE                NOT NULL,
    rate_id     BIGINT              NOT NULL,
    sent        BOOL                NOT NULL,
    last_sent   DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME
);


CREATE INDEX IF NOT EXISTS receipts_building_id_idx ON receipts (building_id);
CREATE INDEX IF NOT EXISTS receipts_month_idx ON receipts (month);
CREATE INDEX IF NOT EXISTS receipts_year_idx ON receipts (year);
CREATE INDEX IF NOT EXISTS receipts_date_idx ON receipts (date);

CREATE TRIGGER IF NOT EXISTS receipts_updated_at_trigger
    AFTER UPDATE
                                ON receipts
                                FOR EACH ROW
BEGIN
UPDATE receipts
SET updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
END;


-- DROP TABLE IF EXISTS expenses;
CREATE TABLE IF NOT EXISTS expenses
(
    id           INTEGER PRIMARY KEY,
    building_id  CHAR(20)                                      NOT NULL,
    receipt_id   VARCHAR(50)                                   NOT NULL,
    description  TEXT                                          NOT NULL,
    amount       DECIMAL(16, 2)                                NOT NULL,
    currency     TEXT CHECK ( currency IN ('USD', 'VED') )     NOT NULL,
    type         TEXT CHECK ( type IN ('COMMON', 'UNCOMMON') ) NOT NULL
    );


CREATE INDEX IF NOT EXISTS expenses_building_id_idx ON expenses (building_id);
CREATE INDEX IF NOT EXISTS expenses_receipt_id_idx ON expenses (receipt_id);


-- DROP TABLE IF EXISTS debts;
CREATE TABLE IF NOT EXISTS debts
(
    building_id                      CHAR(20)       NOT NULL,
    receipt_id                       VARCHAR(50)    NOT NULL,
    apt_number                       CHAR(20)       NOT NULL,
    receipts                         SMALLINT       NOT NULL,
    amount                           DECIMAL(16, 2) NOT NULL,
    months                           TEXT           NOT NULL,
    previous_payment_amount          DECIMAL(16, 2) NOT NULL,
    previous_payment_amount_currency TEXT CHECK ( previous_payment_amount_currency IN ('USD', 'VED') ) NOT NULL,
    PRIMARY KEY (building_id, receipt_id, apt_number)
);



CREATE TABLE IF NOT EXISTS users
(

    id            VARCHAR(50)                             NOT NULL,
    provider_id   VARCHAR(100)                            NOT NULL,
    provider      TEXT CHECK ( provider IN ('PLATFORM', 'GOOGLE', 'GITHUB', 'MASTODON', 'MICROSOFT', 'APPLE',
                               'FACEBOOK') ) NOT NULL,
    email         VARCHAR(320)                            NOT NULL,
    username      VARCHAR(100)                            NOT NULL,
    name          VARCHAR(200)                            NOT NULL,
    picture       VARCHAR(500)                            NOT NULL,
    data          JSONB                                   NOT NULL,
    notification_events TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

-- ALTER TABLE users ADD COLUMN notification_events TEXT;

CREATE INDEX IF NOT EXISTS users_provider_id_idx ON users (provider, provider_id);
CREATE INDEX IF NOT EXISTS users_notification_events_idx ON users (notification_events);


CREATE TABLE IF NOT EXISTS roles  (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER,
    permission_id INTEGER,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id)
);

-- DROP TABLE IF EXISTS user_roles;
CREATE TABLE IF NOT EXISTS user_roles (
    user_id VARCHAR(50),
    role_id INTEGER,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE IF NOT EXISTS telegram_chats (
    user_id VARCHAR(50),
    chat_id BIGINT,
    username VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    pictures TEXT,
    PRIMARY KEY (user_id, chat_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ALTER TABLE telegram_chats ADD COLUMN pictures TEXT;

CREATE TABLE IF NOT EXISTS bcv_files (
    link TEXT PRIMARY KEY,
    rate_count BIGINT NOT NULL,
    sheet_count INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    fileDate DATETIME NOT NULL,
    etag TEXT NOT NULL,
    last_modified TEXT NOT NULL,
    processed_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS bcv_files_created_at_idx ON bcv_files (created_at);