package models

import "time"

type User struct {
	UserID    string    `db:"user_id" json:"user_id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Phone     string    `db:"phone" json:"phone"`
	Password  *string   `db:"password" json:"-"`
	Role      string    `db:"role" json:"role"`
	Verified  bool      `db:"verified" json:"verified"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
