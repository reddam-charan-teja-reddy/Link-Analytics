package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/charan/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

// parseTimeRange reads YYYY-MM-DD query params and falls back to last 30 days.
func (h *AnalyticsHandler) parseTimeRange(c *gin.Context) (time.Time, time.Time) {
	from, err := time.Parse("2006-01-02", c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02")))
	if err != nil {
		from = time.Now().AddDate(0, -1, 0)
	}
	to, err := time.Parse("2006-01-02", c.DefaultQuery("to", time.Now().Format("2006-01-02")))
	if err != nil {
		to = time.Now()
	}
	// Extend 'to' to end of day
	to = to.Add(24*time.Hour - time.Nanosecond)
	return from, to
}

func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)

	summary, err := h.analyticsService.GetSummary(c.Request.Context(), userID, linkID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *AnalyticsHandler) GetClickTimeSeries(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)
	granularity := c.DefaultQuery("granularity", "day")

	points, err := h.analyticsService.GetClickTimeSeries(c.Request.Context(), userID, linkID, from, to, granularity)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if points == nil {
		c.JSON(http.StatusOK, []struct{}{})
		return
	}

	c.JSON(http.StatusOK, points)
}

func (h *AnalyticsHandler) GetSourceBreakdown(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)

	items, err := h.analyticsService.GetSourceBreakdown(c.Request.Context(), userID, linkID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *AnalyticsHandler) GetReferrerBreakdown(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)

	items, err := h.analyticsService.GetReferrerBreakdown(c.Request.Context(), userID, linkID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *AnalyticsHandler) GetLocationBreakdown(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)

	items, err := h.analyticsService.GetLocationBreakdown(c.Request.Context(), userID, linkID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *AnalyticsHandler) GetBrowserBreakdown(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	from, to := h.parseTimeRange(c)

	items, err := h.analyticsService.GetBrowserBreakdown(c.Request.Context(), userID, linkID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *AnalyticsHandler) GetRecentActivity(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	limitParam := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	items, err := h.analyticsService.GetRecentActivity(c.Request.Context(), userID, linkID, limit)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}
