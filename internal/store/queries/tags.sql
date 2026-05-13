-- name: DeleteTagsByID :exec
DELETE FROM issue_tags WHERE id = ?;

-- name: InsertTag :exec
INSERT INTO issue_tags (id, tag) VALUES (?, ?);

-- name: GetUniqueTags :many
SELECT DISTINCT tag FROM issue_tags ORDER BY tag ASC;
