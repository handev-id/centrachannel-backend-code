package exception

import "fmt"

type AppException struct {
    Code    int
    Message string
    Details interface{}
}

func (e *AppException) Error() string {
    return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}

func NewAppException(code int, message string, details interface{}) *AppException {
    return &AppException{
        Code:    code,
        Message: message,
        Details: details,
    }
}

func BadRequest(message string) *AppException {
    return NewAppException(400, message, nil)
}

func Unauthorized(message string) *AppException {
    return NewAppException(401, message, nil)
}

func Forbidden(message string) *AppException {
    return NewAppException(403, message, nil)
}

func NotFound(message string) *AppException {
    return NewAppException(404, message, nil)
}

func InternalServerError(message string) *AppException {
    return NewAppException(500, message, nil)
}
