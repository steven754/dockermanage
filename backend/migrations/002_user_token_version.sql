-- 令牌版本：改密后自增，使改密前签发的所有令牌立即失效
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
