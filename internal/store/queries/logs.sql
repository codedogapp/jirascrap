-- name: InsertLog :exec
INSERT INTO logs (level, message) VALUES (?, ?);
