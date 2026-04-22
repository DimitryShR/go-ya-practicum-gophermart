INSERT INTO users (login, password_hash)
VALUES ($1, $2)
RETURNING id, login, password_hash, current_balance, withdrawn, created_at;
