package api

type SignupParams struct {
	Name     string `db:"name" json:"name"`
	Email    string `db:"email" json:"email"`
	Phone    string `db:"phone" json:"phone"`
	Role     string `db:"role" json:"role"`
	Password string `db:"password" json:"password"`
}

type SignupResponse struct {
	//success code, usually 200
	Code int `json:"code"`

	//Username of the user
	Username string `json:"username"`

	//Message to be displayed
	Message string `json:"message"`
}
