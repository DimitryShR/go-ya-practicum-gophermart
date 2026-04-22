SELECT number, user_id, status, accrual, uploaded_at, updated_at
FROM orders
WHERE number = $1;
