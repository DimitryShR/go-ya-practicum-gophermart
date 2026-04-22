SELECT current_balance, withdrawn
FROM users
WHERE id = $1;
