package database

func migrations(driver string) []string {
	if driver == "postgres" {
		return postgresMigrations()
	}
	return sqliteMigrations()
}

func sqliteMigrations() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry REAL NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry)`,
		`CREATE TABLE IF NOT EXISTS sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL,
			config TEXT NOT NULL DEFAULT '{}',
			enabled INTEGER NOT NULL DEFAULT 1,
			last_fetched_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, type, url)
		)`,
		`CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			source_id INTEGER REFERENCES sources(id) ON DELETE SET NULL,
			guid TEXT NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			published_at TIMESTAMP,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, guid)
		)`,
		`CREATE TABLE IF NOT EXISTS digests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			date TEXT NOT NULL,
			is_auto INTEGER NOT NULL DEFAULT 1,
			articles_reviewed INTEGER DEFAULT 0,
			articles_surfaced INTEGER DEFAULT 0,
			feeds_count INTEGER DEFAULT 0,
			generation_model TEXT DEFAULT '',
			generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			raw_response TEXT DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digests_user_date ON digests(user_id, date, is_auto)`,
		`CREATE TABLE IF NOT EXISTS digest_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			digest_id INTEGER NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
			article_id INTEGER REFERENCES articles(id) ON DELETE SET NULL,
			position INTEGER NOT NULL DEFAULT 0,
			priority INTEGER NOT NULL DEFAULT 5,
			category TEXT NOT NULL DEFAULT '',
			headline TEXT NOT NULL,
			tldr TEXT NOT NULL DEFAULT '',
			bullets TEXT NOT NULL DEFAULT '[]',
			source_name TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			read_time INTEGER NOT NULL DEFAULT 0,
			language TEXT NOT NULL DEFAULT '',
			importance TEXT NOT NULL DEFAULT 'medium'
		)`,
		`CREATE TABLE IF NOT EXISTS custom_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			prompt TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS section_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			digest_id INTEGER NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
			section_id INTEGER NOT NULL REFERENCES custom_sections(id) ON DELETE CASCADE,
			article_id INTEGER REFERENCES articles(id) ON DELETE SET NULL,
			position INTEGER NOT NULL DEFAULT 0,
			headline TEXT NOT NULL,
			tldr TEXT NOT NULL DEFAULT '',
			bullets TEXT NOT NULL DEFAULT '[]',
			source_name TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS interests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			label TEXT NOT NULL DEFAULT '',
			value TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS votes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			digest_item_id INTEGER NOT NULL REFERENCES digest_items(id) ON DELETE CASCADE,
			value INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, digest_item_id)
		)`,
	}
}

func postgresMigrations() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BYTEA NOT NULL,
			expiry TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry)`,
		`CREATE TABLE IF NOT EXISTS sources (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL,
			config JSONB NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			last_fetched_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, type, url)
		)`,
		`CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			source_id BIGINT REFERENCES sources(id) ON DELETE SET NULL,
			guid TEXT NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			published_at TIMESTAMPTZ,
			fetched_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, guid)
		)`,
		`CREATE TABLE IF NOT EXISTS digests (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			date TEXT NOT NULL,
			is_auto BOOLEAN NOT NULL DEFAULT TRUE,
			articles_reviewed INT DEFAULT 0,
			articles_surfaced INT DEFAULT 0,
			feeds_count INT DEFAULT 0,
			generation_model TEXT DEFAULT '',
			generated_at TIMESTAMPTZ DEFAULT NOW(),
			raw_response TEXT DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digests_user_date ON digests(user_id, date, is_auto)`,
		`CREATE TABLE IF NOT EXISTS digest_items (
			id BIGSERIAL PRIMARY KEY,
			digest_id BIGINT NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
			article_id BIGINT REFERENCES articles(id) ON DELETE SET NULL,
			position INT NOT NULL DEFAULT 0,
			priority INT NOT NULL DEFAULT 5,
			category TEXT NOT NULL DEFAULT '',
			headline TEXT NOT NULL,
			tldr TEXT NOT NULL DEFAULT '',
			bullets JSONB NOT NULL DEFAULT '[]',
			source_name TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			read_time INT NOT NULL DEFAULT 0,
			language TEXT NOT NULL DEFAULT '',
			importance TEXT NOT NULL DEFAULT 'medium'
		)`,
		`CREATE TABLE IF NOT EXISTS custom_sections (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			prompt TEXT NOT NULL,
			position INT NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS section_items (
			id BIGSERIAL PRIMARY KEY,
			digest_id BIGINT NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
			section_id BIGINT NOT NULL REFERENCES custom_sections(id) ON DELETE CASCADE,
			article_id BIGINT REFERENCES articles(id) ON DELETE SET NULL,
			position INT NOT NULL DEFAULT 0,
			headline TEXT NOT NULL,
			tldr TEXT NOT NULL DEFAULT '',
			bullets JSONB NOT NULL DEFAULT '[]',
			source_name TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS interests (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			label TEXT NOT NULL DEFAULT '',
			value TEXT NOT NULL,
			position INT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS votes (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			digest_item_id BIGINT NOT NULL REFERENCES digest_items(id) ON DELETE CASCADE,
			value INT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, digest_item_id)
		)`,
	}
}
