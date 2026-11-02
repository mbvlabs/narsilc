CREATE TABLE users (id integer NOT NULL, email text NOT NULL, name text);
CREATE TABLE posts (id integer NOT NULL, user_id integer NOT NULL, title text NOT NULL);
