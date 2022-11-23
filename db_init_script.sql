DROP TABLE IF EXISTS relay_counts;
DROP TABLE IF EXISTS todays_relay_counts;
DROP TABLE IF EXISTS daily_app_sums;
DROP TABLE IF EXISTS todays_app_sums;
CREATE TABLE relay_counts (
  id INT GENERATED ALWAYS AS IDENTITY,
  origin VARCHAR NOT NULL,
  application VARCHAR,
  count bigint,
  count_success INT,
  count_failure INT,
  time TIMESTAMPTZ
);
CREATE TABLE todays_relay_counts (
  id INT GENERATED ALWAYS AS IDENTITY,
  time TIMESTAMPTZ,
  application VARCHAR,
  origin VARCHAR NOT NULL,
  count_success INT,
  count_failure INT,
  count bigint
);
CREATE TABLE daily_app_sums (
  id INT GENERATED ALWAYS AS IDENTITY,
  application VARCHAR NOT NULL,
  count bigint,
  time TIMESTAMPTZ,
  count_success INT,
  count_failure INT
);
CREATE TABLE todays_app_sums (
  id INT GENERATED ALWAYS AS IDENTITY,
  application VARCHAR NOT NULL,
  count_success INT,
  count_failure INT,
  count bigint
);