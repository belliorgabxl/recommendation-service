CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    age INT NOT NULL CHECK (age > 0),
    country CHAR(2) NOT NULL CHECK (country ~ '^[A-Z]{2}$'),
    subscription_type VARCHAR(20) NOT NULL CHECK (subscription_type IN ('free', 'basic', 'premium')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS content (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    genre VARCHAR(50) NOT NULL,
    popularity_score DOUBLE PRECISION NOT NULL CHECK (popularity_score >= 0 AND popularity_score <= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_watch_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id BIGINT NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    watched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_country
    ON users(country);

CREATE INDEX IF NOT EXISTS idx_users_subscription
    ON users(subscription_type);

CREATE INDEX IF NOT EXISTS idx_content_genre
    ON content(genre);

CREATE INDEX IF NOT EXISTS idx_content_popularity
    ON content(popularity_score DESC);

CREATE INDEX IF NOT EXISTS idx_watch_history_user
    ON user_watch_history(user_id);

CREATE INDEX IF NOT EXISTS idx_watch_history_content
    ON user_watch_history(content_id);

CREATE INDEX IF NOT EXISTS idx_watch_history_user_watched_at
    ON user_watch_history(user_id, watched_at DESC);