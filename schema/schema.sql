-- DROP TABLE IF EXISTS rates;

CREATE TABLE IF NOT EXISTS rates
(
    id            BIGINT PRIMARY KEY,
    from_currency TEXT                                           NOT NULL,
    to_currency   TEXT                                           NOT NULL,
    rate          DECIMAL(16, 12)                                NOT NULL,
    source        TEXT                                           NOT NULL,
    date_of_rate  DATE                                           NOT NULL,
    date_of_file  DATE                                           NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    etag          varchar(20),
    last_modified varchar(40)
);

CREATE INDEX IF NOT EXISTS rates_from_currency_to_currency_rate_date_of_rate_idx
    ON rates (from_currency, to_currency, rate, date_of_rate);

CREATE INDEX IF NOT EXISTS rates_from_currency_idx ON rates (from_currency);
CREATE INDEX IF NOT EXISTS rates_date_of_rate_idx ON rates (date_of_rate);
CREATE INDEX IF NOT EXISTS rates_from_currency_date_of_rate_idx ON rates (from_currency, date_of_rate);



CREATE TABLE IF NOT EXISTS apartments
(
    building_id CHAR(20)      NOT NULL,
    number      CHAR(20)      NOT NULL,
    name        VARCHAR(100)  NOT NULL,
    id_doc      CHAR(20),
    aliquot     DECIMAL(3, 2) NOT NULL DEFAULT 0.00,
    emails      TEXT,
    created_at  DATETIME               DEFAULT CURRENT_TIMESTAMP,
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

CREATE TABLE IF NOT EXISTS buildings
(
    id                               CHAR(20)                                       NOT NULL UNIQUE PRIMARY KEY,
    name                             VARCHAR(100)                                   NOT NULL,
    rif                              CHAR(20)                                       NOT NULL,
    main_currency                    TEXT CHECK ( main_currency IN ('USD', 'VED') ) NOT NULL,
    debt_currency                    TEXT CHECK ( debt_currency IN ('USD', 'VED') ) NOT NULL,
    currencies_to_show_amount_to_pay TEXT                                           NOT NULL,
    fixed_pay                        BOOL                                           NOT NULL,
    fixed_pay_amount                 DECIMAL(16, 2),
    round_up_payments                BOOL                                           NOT NULL,
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
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME
    );


CREATE INDEX IF NOT EXISTS extra_charges_parent_reference_idx ON extra_charges (parent_reference);
CREATE INDEX IF NOT EXISTS extra_charges_building_id_idx ON extra_charges (building_id);
CREATE INDEX IF NOT EXISTS extra_charges_parent_reference_building_id_idx ON extra_charges (parent_reference, building_id);
CREATE INDEX IF NOT EXISTS extra_charges_type_idx ON extra_charges (type);

CREATE TRIGGER IF NOT EXISTS extra_charges_updated_at_trigger
    AFTER UPDATE
                                ON extra_charges
                                FOR EACH ROW
BEGIN
UPDATE extra_charges
SET updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
END;


CREATE TABLE IF NOT EXISTS reserve_funds
(
    id              INTEGER PRIMARY KEY,
    building_id     CHAR(20)                                              NOT NULL,
    name            VARCHAR(100)                                          NOT NULL,
    fund            DECIMAL(16, 2)                                        NOT NULL,
    expense         DECIMAL(16, 2),
    pay             DECIMAL(16, 2),
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

