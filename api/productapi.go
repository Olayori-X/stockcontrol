package api

type AddProductInput struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
}

type GetProductBySKUInput struct {
	SKU string `json:"sku"`
}

type EditProductInput struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
}

type DeleteProductInput struct {
	SKU string `json:"sku"`
}
