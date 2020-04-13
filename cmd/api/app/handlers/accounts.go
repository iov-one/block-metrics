package handlers

import (
	"net/http"

	"github.com/iov-one/block-metrics/pkg/models"

	"github.com/iov-one/block-metrics/pkg/store"
	"github.com/labstack/echo/v4"
)

type AccountsHandler struct {
	Store *store.Store
}

func (h *AccountsHandler) GetAccount(c echo.Context) error {
	name := c.QueryParam("name")
	domain := c.QueryParam("domain")
	if domain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide valid domain")
	}

	accs, err := h.Store.LoadAccount(c.Request().Context(), name, domain)
	if err != nil {
		return err
	}

	targets, err := h.Store.LoadAccountTargets(c.Request().Context(), name, domain)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK,
		struct {
			Account *models.Account        `json:"account"`
			Targets []models.AccountTarget `json:"targets"`
		}{
			accs,
			targets,
		})
}
