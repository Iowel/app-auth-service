CREATE TABLE IF NOT EXISTS user_stat (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    event_description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);