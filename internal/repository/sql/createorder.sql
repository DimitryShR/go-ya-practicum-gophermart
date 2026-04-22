WITH inserted AS (
    INSERT INTO orders (number, user_id, status)
    VALUES ($1, $2, $3)
    ON CONFLICT (number) DO NOTHING
    RETURNING number, user_id, status, accrual, uploaded_at, updated_at, TRUE AS inserted
)
SELECT number, user_id, status, accrual, uploaded_at, updated_at, inserted
FROM inserted
UNION ALL
SELECT number, user_id, status, accrual, uploaded_at, updated_at, FALSE AS inserted
FROM orders
WHERE number = $1
LIMIT 1;
