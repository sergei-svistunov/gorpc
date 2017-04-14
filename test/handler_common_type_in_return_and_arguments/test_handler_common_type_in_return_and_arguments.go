package handler_common_type_in_different_versions_return_values

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Caption() string {
	return "Test handler 1"
}

func (h *Handler) Description() string {
	return "Handler for tests"
}
