-- name: GetTodosByTicket :many
SELECT id, title, done FROM issue_todos WHERE ticket_id = ? ORDER BY id ASC;

-- name: DeleteTodosByTicket :exec
DELETE FROM issue_todos WHERE ticket_id = ?;

-- name: InsertTodo :exec
INSERT INTO issue_todos (ticket_id, title, done) VALUES (?, ?, ?);
