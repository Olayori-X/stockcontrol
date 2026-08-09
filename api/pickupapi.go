package api

import "github.com/Olayori-X/stock-control-backend/models"

type CreatePickupRequestInput struct {
	SalesAssociateID string               `json:"sales_associate_id"`
	DistributorID    string               `json:"distributor_id"`
	Products         []models.ProductItem `json:"products"`
}

type ConfirmPickupRequestInput struct {
	RequestID     string `json:"request_id"`
	DistributorID string `json:"distributor_id"`
}

type SearchDistributorsResponse struct {
	Users []DistributorSearchResult `json:"users"`
	Total int                       `json:"total"`
}

type DistributorSearchResult struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Role   string `json:"role"`
}

// ToSearchResult converts a models.User to the safe public shape.
func ToSearchDistributorResult(u models.User) DistributorSearchResult {
	return DistributorSearchResult{
		UserID: u.UserID,
		Name:   u.Name,
		Email:  u.Email,
		Phone:  u.Phone,
		Role:   u.Role,
	}
}
