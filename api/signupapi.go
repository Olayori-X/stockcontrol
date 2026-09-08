package api

type SignupParams struct {
	Name     string `db:"name" json:"name"`
	Email    string `db:"email" json:"email"`
	Phone    string `db:"phone" json:"phone"`
	Role     string `db:"role" json:"role"`
	Password string `db:"password" json:"password"`
}

type SignupResponse struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	PIN    string `json:"pin,omitempty"` // present only when role=sales
}

type SetPINInput struct {
	UserID string `json:"user_id"`
}

type SetPINResponse struct {
	UserID string `json:"user_id"`
	PIN    string `json:"pin"` // shown once, at creation/reset time only
}

type PINLoginInput struct {
	UserID    string   `json:"user_id"`
	PIN       string   `json:"pin"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	DeviceRef string   `json:"device_ref,omitempty"`
}
