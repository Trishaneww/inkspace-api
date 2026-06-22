package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetDashboard(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	data, err := h.svc.GetDashboard(c.Request.Context(), userID, c.Query("range"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "dashboard_failed", "failed to load dashboard")
		return
	}
	httpx.OK(c, data)
}
