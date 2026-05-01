CREATE TABLE preview_rices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL UNIQUE,
    price NUMERIC(5, 2) CHECK((price IS NULL) OR (price IS NOT NULL AND price > 0.0)),
    thumbnail_path TEXT NOT NULL UNIQUE,
    download_count INT NOT NULL DEFAULT 0 CHECK(download_count >= 0),
    star_count INT NOT NULL DEFAULT 0 CHECK(star_count >= 0),
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);