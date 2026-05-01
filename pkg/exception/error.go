package exception

// ApiError interface unify error
type ApiError struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误描述
}

func (e *ApiError) Error() string {
	return e.Message
}

// NewApiError create ApiError
func NewApiError(code int) *ApiError {
	return &ApiError{
		Code:    code,
		Message: GetResultMsg(code),
	}
}
