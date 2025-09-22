package utils

// Используется для возвращения ошибки пользователю
type ErrorResponse struct {
	// Описание ошибки в свободной форме
	Reason string `json:"reason"`
}
