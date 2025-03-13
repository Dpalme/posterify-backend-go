CREATE TABLE IF NOT EXISTS users(
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_index ON users (email);

CREATE TABLE IF NOT EXISTS collections(
    id SERIAL PRIMARY KEY,
    author_id INTEGER NOT NULL,
    name VARCHAR(64) NOT NULL,
    description VARCHAR(255),
    poster VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_name_author UNIQUE (author_id, name),
    CONSTRAINT fk_user FOREIGN KEY(author_id) REFERENCES users(id) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS collections_images (
    img_path VARCHAR(64) NOT NULL,
    collection_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (img_path, collection_id),
    CONSTRAINT fk_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE
);

CREATE INDEX idx_collection_id ON collections_images (collection_id);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = now();
   RETURN NEW;
END;
$$ language plpgsql;

CREATE TRIGGER update_users_updated_at BEFORE UPDATE
ON users FOR EACH ROW EXECUTE PROCEDURE
update_updated_at_column();

CREATE TRIGGER update_collections_updated_at BEFORE UPDATE
ON collections FOR EACH ROW EXECUTE PROCEDURE
update_updated_at_column();
