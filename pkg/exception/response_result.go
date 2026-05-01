package exception

// ResponseResult 统一返回结果
type ResponseResult struct {
	Code    string      `json:"code"`    // 状态码
	Message string      `json:"message"` // 描述
	Body    interface{} `json:"body"`    // 数据体
}

func NewResponseResult(code, message string, body interface{}) *ResponseResult {
	return &ResponseResult{code, message, body}
}
