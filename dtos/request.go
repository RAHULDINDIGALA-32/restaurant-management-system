package dtos

type UpdateFoodRequest struct {
	Name    *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Price   *float64 `json:"price"`
	Image   *string  `json:"image"`
	Menu_id *string  `json:"menu_id"`
}
