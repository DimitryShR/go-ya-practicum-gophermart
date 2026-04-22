SELECT id, login, password_hash, current_balance, withdrawn, created_at
FROM users
WHERE login = $1;
