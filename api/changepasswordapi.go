package api

type ChangePasswordParams struct {
	Email           string `db:"email" json:"email"`
	Password        string `db:"password" json:"password"`
	ConfirmPassword string `db:"password" json:"confirm_password"`
}

type ChangePasswordResponse struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
}
