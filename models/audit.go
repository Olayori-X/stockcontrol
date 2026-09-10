package models

import "time"

type AuditLogEntry struct {
	ActorID   *string   `db:"actor_id" json:"actor_id"` // nil = system-initiated
	Action    string    `db:"action" json:"action"`
	Target    string    `db:"target" json:"target"`
	Details   string    `db:"details" json:"details"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
