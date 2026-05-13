-- name: DeleteTopLevelTickets :exec
DELETE FROM tickets WHERE epic_id IS NULL;

-- name: UpsertTicket :exec
INSERT OR REPLACE INTO tickets (
  id, summary, reporter, status, status_category,
  priority, type, created_at, updated_at, markdown, epic_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCachedTickets :many
SELECT t.id, t.summary, t.reporter, t.status, t.status_category,
       t.priority, t.type, t.created_at, t.updated_at, t.markdown,
       COALESCE(GROUP_CONCAT(it.tag), '') AS tags
FROM tickets t
LEFT JOIN issue_tags it ON t.id = it.id
WHERE t.epic_id IS NULL
   OR LOWER(t.type) = 'epic'
GROUP BY t.id
ORDER BY t.updated_at DESC;

-- name: DeleteEpicChildren :exec
DELETE FROM tickets WHERE epic_id = ?;

-- name: GetAllEpicChildren :many
SELECT t.epic_id,
       t.id,
       t.summary,
       t.reporter,
       t.status,
       t.status_category,
       t.priority,
       t.type,
       t.created_at,
       t.updated_at,
       t.markdown,
       COALESCE(GROUP_CONCAT(it.tag), '') AS tags
FROM tickets t
LEFT JOIN issue_tags it ON t.id = it.id
WHERE t.epic_id IS NOT NULL
GROUP BY t.id
ORDER BY t.epic_id, t.updated_at DESC;
