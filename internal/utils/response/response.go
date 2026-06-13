package response

import (
    "github.com/gofiber/fiber/v3"
)

type ResponseMeta struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type Response struct {
    Meta   ResponseMeta `json:"meta"`
    Data   interface{}  `json:"data,omitempty"`
    Errors interface{}  `json:"errors,omitempty"`
}

func Success(c fiber.Ctx, statusCode int, message string, data interface{}) error {
    return c.Status(statusCode).JSON(Response{
        Meta: ResponseMeta{
            Code:    statusCode,
            Message: message,
        },
        Data: data,
    })
}

func Error(c fiber.Ctx, statusCode int, message string, errors interface{}) error {
    return c.Status(statusCode).JSON(Response{
        Meta: ResponseMeta{
            Code:    statusCode,
            Message: message,
        },
        Errors: errors,
    })
}

func Created(c fiber.Ctx, message string, data interface{}) error {
    return Success(c, fiber.StatusCreated, message, data)
}

func OK(c fiber.Ctx, message string, data interface{}) error {
    return Success(c, fiber.StatusOK, message, data)
}

func BadRequest(c fiber.Ctx, message string, errors interface{}) error {
    return Error(c, fiber.StatusBadRequest, message, errors)
}

func Unauthorized(c fiber.Ctx, message string) error {
    return Error(c, fiber.StatusUnauthorized, message, nil)
}

func Forbidden(c fiber.Ctx, message string) error {
    return Error(c, fiber.StatusForbidden, message, nil)
}

func NotFound(c fiber.Ctx, message string) error {
    return Error(c, fiber.StatusNotFound, message, nil)
}

func InternalServerError(c fiber.Ctx, message string) error {
    return Error(c, fiber.StatusInternalServerError, message, nil)
}
