SELECT order_number, user_id, sum, processed_at
FROM withdrawals
WHERE user_id = $1
ORDER BY processed_at DESC;
