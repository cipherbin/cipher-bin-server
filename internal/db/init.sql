CREATE TABLE messages (
  id serial PRIMARY KEY,
  uuid uuid NOT NULL,
  message text NOT NULL
);

ALTER TABLE messages
  ADD COLUMN email varchar(255) DEFAULT '',
  ADD COLUMN reference_name varchar(255) DEFAULT '';

ALTER TABLE messages
  ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();

ALTER TABLE messages
  ADD COLUMN password varchar(255) DEFAULT '';
