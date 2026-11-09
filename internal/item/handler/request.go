package handler

// CreateItemRequest define el payload para la creación de un ítem.
type CreateItemRequest struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}

// UpdateStatusRequest define el payload para la transición de estado.
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=PENDING ACTIVE ARCHIVED"`
}
