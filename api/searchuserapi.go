package api

import "github.com/Olayori-X/stock-control-backend/models"

// SearchUsersResponse is what the handler encodes back to the client.
// We return a safe subset of models.User — no email, phone, password, or code.
type SearchUsersResponse struct {
	Users []UserSearchResult `json:"users"`
	Total int                `json:"total"`
}

type UserSearchResult struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Role   string `json:"role"`
}

// ToSearchResult converts a models.User to the safe public shape.
func ToSearchResult(u models.User) UserSearchResult {
	return UserSearchResult{
		UserID: u.UserID,
		Name:   u.Name,
		Email:  u.Email,
		Phone:  u.Phone,
		Role:   u.Role,
	}
}

type UserSummary struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	Verified  bool   `json:"verified"`
	CreatedAt string `json:"created_at"`
}

type GroupedUsersResponse struct {
	Admins       []UserSummary `json:"admins"`
	Sales        []UserSummary `json:"sales"`
	Distributors []UserSummary `json:"distributors"`
}

func ToUserSummary(u models.User) UserSummary {
	return UserSummary{
		UserID:    u.UserID,
		Name:      u.Name,
		Email:     u.Email,
		Phone:     u.Phone,
		Role:      u.Role,
		Verified:  true,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
