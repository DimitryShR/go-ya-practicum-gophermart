WITH selected AS (
    SELECT number
    FROM orders
    WHERE status IN ($1, $2)
    ORDER BY updated_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
UPDATE orders AS o
SET updated_at = NOW()
FROM selected
WHERE o.number = selected.number
RETURNING o.number, o.user_id, o.status, o.accrual, o.uploaded_at, o.updated_at;
