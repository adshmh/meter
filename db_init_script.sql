DROP TABLE IF EXISTS relay_counts;
DROP TABLE IF EXISTS daily_app_sums;
DROP TABLE IF EXISTS todays_app_sums;
CREATE TABLE relay_counts (
  id INT GENERATED ALWAYS AS IDENTITY,
  application VARCHAR NOT NULL,
  count bigint NOT NULL,
  time TIMESTAMPTZ
);
CREATE TABLE daily_app_sums (
  id INT GENERATED ALWAYS AS IDENTITY,
  application VARCHAR NOT NULL,
  count bigint NOT NULL,
  time TIMESTAMPTZ
);
CREATE TABLE todays_app_sums (
  id INT GENERATED ALWAYS AS IDENTITY,
  application VARCHAR NOT NULL,
  count bigint NOT NULL
);