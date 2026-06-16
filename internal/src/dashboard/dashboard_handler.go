package dashboard

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/tenant"
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

func NewDashboardHandlerWithService(service DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) Stats(c fiber.Ctx, t *tenant.Tenant) error {
	days, _ := strconv.Atoi(c.Query("days", "30"))

	stats, err := h.service.GetStats(c.Context(), t.ID, days)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", stats)
}

func (h *DashboardHandler) Chart(c fiber.Ctx, t *tenant.Tenant) error {
	days, _ := strconv.Atoi(c.Query("days", "7"))

	chart, err := h.service.GetChart(c.Context(), t.ID, days)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", chart)
}
