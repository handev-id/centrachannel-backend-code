package dashboard

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type DashboardHandler struct {
	service DashboardService
}

func NewDashboardHandler(c *di.Container) *DashboardHandler {
	repo := NewDashboardRepository()
	service := NewDashboardService(repo, c.DB)
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) Stats(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	stats, err := h.service.GetStats(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", stats)
}

func (h *DashboardHandler) Chart(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	days, _ := strconv.Atoi(c.Query("days", "30"))

	chart, err := h.service.GetChart(c.Context(), t.ID, days)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", chart)
}
