package api

type AssignDistributorInput struct {
	SalesAssociateID string `json:"sales_associate_id"`
	DistributorID    string `json:"distributor_id"`
}
