package response

import (
    "github.com/go-playground/validator/v10"
    "github.com/gofiber/fiber/v3"
)

var validate = validator.New()

func Validate(c fiber.Ctx, req interface{}) error {
	if err := validate.Struct(req); err != nil {
		return UnprocessableEntity(c, "Validation failed", formatValidationErrors(err))
	}
	return nil
}

func UnprocessableEntity(c fiber.Ctx, message string, errors interface{}) error {
	return Error(c, fiber.StatusUnprocessableEntity, message, errors)
}

func formatValidationErrors(err error) map[string]string {
    errors := make(map[string]string)
    if verrs, ok := err.(validator.ValidationErrors); ok {
        for _, fe := range verrs {
            errors[fe.Field()] = fe.Tag()
        }
    }
    return errors
}

type ErrorBody struct {
    Message string      `json:"message,omitempty"`
    Errors  interface{} `json:"errors,omitempty"`
}

type ResponseMeta struct {
    Message     string `json:"message"`
    Total       *int   `json:"total,omitempty"`
    PerPage     *int   `json:"per_page,omitempty"`
    CurrentPage *int   `json:"current_page,omitempty"`
    LastPage    *int   `json:"last_page,omitempty"`
    From        *int   `json:"from,omitempty"`
    To          *int   `json:"to,omitempty"`
    LastID      *int   `json:"last_id,omitempty"`
    LastActivity string `json:"last_activity,omitempty"`
    HasMore     *bool  `json:"has_more,omitempty"`
}

type Response struct {
    Meta ResponseMeta `json:"meta"`
    Data interface{}  `json:"data"`
}

func Paginated(c fiber.Ctx, message string, data interface{}, total, page, limit, from, to, lastPage int) error {
    t, p, l, f, t2, lp := total, page, limit, from, to, lastPage
    return c.Status(fiber.StatusOK).JSON(Response{
        Meta: ResponseMeta{
            Message:     message,
            Total:       &t,
            PerPage:     &p,
            CurrentPage: &l,
            LastPage:    &lp,
            From:        &f,
            To:          &t2,
        },
        Data: data,
    })
}

func CursorPaginated(c fiber.Ctx, message string, data interface{}, lastID int, hasMore bool, lastActivity ...string) error {
    li := lastID
    hm := hasMore
    m := ResponseMeta{Message: message, LastID: &li, HasMore: &hm}
    if len(lastActivity) > 0 {
        m.LastActivity = lastActivity[0]
    }
    return c.Status(fiber.StatusOK).JSON(Response{
        Meta: m,
        Data: data,
    })
}

func OK(c fiber.Ctx, message string, data interface{}) error {
    return c.Status(fiber.StatusOK).JSON(Response{
        Meta: ResponseMeta{Message: message},
        Data: data,
    })
}

func Created(c fiber.Ctx, message string, data interface{}) error {
    return c.Status(fiber.StatusCreated).JSON(Response{
        Meta: ResponseMeta{Message: message},
        Data: data,
    })
}

func Error(c fiber.Ctx, statusCode int, message string, errors interface{}) error {
    return c.Status(statusCode).JSON(ErrorBody{
        Message: message,
        Errors:  errors,
    })
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
