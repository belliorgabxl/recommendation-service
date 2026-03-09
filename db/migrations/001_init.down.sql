DROP INDEX IF EXISTS idx_watch_history_user_watched_at;
DROP INDEX IF EXISTS idx_watch_history_content;
DROP INDEX IF EXISTS idx_watch_history_user;
DROP INDEX IF EXISTS idx_content_popularity;
DROP INDEX IF EXISTS idx_content_genre;
DROP INDEX IF EXISTS idx_users_subscription;
DROP INDEX IF EXISTS idx_users_country;

DROP TABLE IF EXISTS user_watch_history;
DROP TABLE IF EXISTS content;
DROP TABLE IF EXISTS users;