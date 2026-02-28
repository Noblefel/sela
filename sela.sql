CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    username VARCHAR(40) UNIQUE NOT NULL,
    bio VARCHAR(250),
    avatar VARCHAR,
    admin BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW() 
);

CREATE TABLE IF NOT EXISTS articles (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id INT NOT NULL,
    title VARCHAR(150) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    excerpt VARCHAR(500),
    content TEXT NOT NULL,
    image VARCHAR,

    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX article_created_idx ON articles(created_at);
 
INSERT INTO users (email, username, name) 
VALUES ('first@test.com', 'first-user', 'FIRST USER')
ON CONFLICT (email) DO NOTHING;

CREATE TABLE IF NOT EXISTS reset_emails (
    token VARCHAR(50) PRIMARY KEY,
    user_id INT NOT NULL, 
    code VARCHAR(6) NOT NULL,
    email VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- CREATE TABLE IF NOT EXISTS likes_article (
--     user_id INT NOT NULL,
--     article_id INT NOT NULL,
--     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

--     PRIMARY KEY(user_id, article_id),
--     FOREIGN KEY(user_id) REFERENCES users(id),
--     FOREIGN KEY(article_id) REFERENCES articles(id)
-- )

-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public to sela; 