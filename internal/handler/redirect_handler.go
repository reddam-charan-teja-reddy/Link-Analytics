package handler

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/charan/url-shortener/internal/service"
	"github.com/charan/url-shortener/internal/worker"
	"github.com/charan/url-shortener/pkg/botdetect"
	"github.com/charan/url-shortener/pkg/useragent"
	"github.com/gin-gonic/gin"
)

type RedirectHandler struct {
	redirectService *service.RedirectService
	clickWorker     *worker.ClickWorker
	botDetector     *botdetect.Detector
	uaParser        *useragent.Parser
}

func NewRedirectHandler(redirectService *service.RedirectService, clickWorker *worker.ClickWorker, botDetector *botdetect.Detector, uaParser *useragent.Parser) *RedirectHandler {
	return &RedirectHandler{
		redirectService: redirectService,
		clickWorker:     clickWorker,
		botDetector:     botDetector,
		uaParser:        uaParser,
	}
}

func (h *RedirectHandler) Redirect(c *gin.Context) {
	hash := c.Param("hash")

	switch hash {
	case "health", "auth", "api", "favicon.ico":
		c.Status(http.StatusNotFound)
		return
	}

	result, err := h.redirectService.Resolve(c.Request.Context(), hash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	sourceName := strings.TrimSpace(c.Query("src"))

	ua := c.GetHeader("User-Agent")
	parsed := h.uaParser.Parse(ua)
	isBot := h.botDetector.IsBot(ua)

	event := domain.ClickEvent{
		Hash:         hash,
		LinkID:       result.LinkID,
		SourceLinkID: result.SourceLinkID,
		SourceName:   sourceName,
		IPAddress:    c.ClientIP(),
		UserAgent:    ua,
		Referer:      c.GetHeader("Referer"),
		Browser:      parsed.Browser,
		OS:           parsed.OS,
		IsBot:        isBot,
		ClickedAt:    time.Now(),
	}

	// fire and forget — don't slow down the redirect
	h.clickWorker.Enqueue(event)

	destination := mergeRedirectQuery(result.OriginalURL, c.Request.URL.Query())
	c.Redirect(http.StatusFound, destination)
}

func mergeRedirectQuery(originalURL string, incoming url.Values) string {
	parsed, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	merged := parsed.Query()
	for key, values := range incoming {
		for _, value := range values {
			merged.Add(key, value)
		}
	}

	parsed.RawQuery = merged.Encode()
	return parsed.String()
}
