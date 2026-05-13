-- +goose Up
CREATE INDEX IF NOT EXISTS idx_issue_todos_ticket_id ON issue_todos(ticket_id);
CREATE INDEX IF NOT EXISTS idx_tickets_epic_id ON tickets(epic_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tickets_epic_id;
DROP INDEX IF EXISTS idx_issue_todos_ticket_id;
