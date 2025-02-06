-- DROP TABLE IF EXISTS rates;

CREATE TABLE IF NOT EXISTS rates
(
    id            INTEGER PRIMARY KEY,
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


-- select *
-- from rates
-- where ((from_currency = 'EUR')
--     AND (to_currency = 'VES')
--     AND (rate = 86238.96585797) AND (date_of_rate = DATE('2020-03-30 00:00:00-04:00')));

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

-- CREATE TABLE IF NOT EXISTS bcv_files
-- (
--     id         INTEGER PRIMARY KEY,
--     url        TEXT    UNIQUE NOT NULL,
--     etag       TEXT,
--     size       INTEGER,
--     hash       INTEGER,
--     last_check DATETIME,
--     created_at DATETIME DEFAULT CURRENT_TIMESTAMP
-- );
--
-- CREATE INDEX IF NOT EXISTS bcv_files_url_dix ON bcv_files (url);