SELECT number, user_id, status, accrual, uploaded_at, updated_at
FROM orders
WHERE user_id = $1
ORDER BY uploaded_at DESC;
