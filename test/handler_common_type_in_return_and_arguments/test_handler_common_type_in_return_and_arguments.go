package handler_common_type_in_return_and_arguments

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
