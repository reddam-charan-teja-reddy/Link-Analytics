package handler

import (
	"net/http"

	"github.com/charan/url-shortener/internal/dto"
	"github.com/charan/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

type LinkHandler struct {
	linkService *service.LinkService
}

func NewLinkHandler(linkService *service.LinkService) *LinkHandler {
	return &LinkHandler{linkService: linkService}
}

func (h *LinkHandler) Create(c *gin.Context) {
	var req dto.CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	link, err := h.linkService.Create(c.Request.Context(), userID, req.OriginalURL, req.Title)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, link)
}

func (h *LinkHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	link, err := h.linkService.Get(c.Request.Context(), userID, linkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	c.JSON(http.StatusOK, link)
}

func (h *LinkHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Query("group_id")
	sourceName := c.Query("source")

	links, err := h.linkService.List(c.Request.Context(), userID, groupID, sourceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if links == nil {
		links = []service.LinkResponse{}
	}

	c.JSON(http.StatusOK, links)
}
func (h *LinkHandler) CreateGroup(c *gin.Context) {
	userID := c.GetString("userID")
	var req dto.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.linkService.CreateGroup(c.Request.Context(), userID, req.Name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *LinkHandler) ListGroups(c *gin.Context) {
	userID := c.GetString("userID")
	items, err := h.linkService.ListGroups(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *LinkHandler) UpdateGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("groupId")
	var req dto.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.linkService.UpdateGroup(c.Request.Context(), userID, groupID, req.Name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *LinkHandler) DeleteGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("groupId")

	if err := h.linkService.DeleteGroup(c.Request.Context(), userID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

func (h *LinkHandler) ListLinkGroups(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	groups, err := h.linkService.ListLinkGroups(c.Request.Context(), userID, linkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (h *LinkHandler) AddLinkToGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("groupId")
	var req dto.AssignGroupLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.linkService.AddLinkToGroup(c.Request.Context(), userID, groupID, req.LinkID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "link added to group"})
}

func (h *LinkHandler) RemoveLinkFromGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("groupId")
	linkID := c.Param("linkId")

	if err := h.linkService.RemoveLinkFromGroup(c.Request.Context(), userID, groupID, linkID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "link removed from group"})
}

func (h *LinkHandler) BatchCreateSources(c *gin.Context) {
	userID := c.GetString("userID")
	var req dto.BatchSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.linkService.BatchCreateSources(c.Request.Context(), userID, req.SourceName, req.ScopeType, req.ScopeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Update partially updates a link. If is_active is omitted, the current state is preserved.
func (h *LinkHandler) Update(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	var req dto.UpdateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.linkService.Get(c.Request.Context(), userID, linkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	link, err := h.linkService.Update(c.Request.Context(), userID, linkID, title, isActive)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	c.JSON(http.StatusOK, link)
}

func (h *LinkHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	if err := h.linkService.Delete(c.Request.Context(), userID, linkID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "link deleted"})
}

func (h *LinkHandler) CreateSource(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	var req dto.CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source, err := h.linkService.CreateSource(c.Request.Context(), userID, linkID, req.SourceName)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, source)
}

func (h *LinkHandler) ListSources(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")

	sources, err := h.linkService.ListSources(c.Request.Context(), userID, linkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	if sources == nil {
		sources = []service.SourceLinkResponse{}
	}

	c.JSON(http.StatusOK, sources)
}

func (h *LinkHandler) DeleteSource(c *gin.Context) {
	userID := c.GetString("userID")
	linkID := c.Param("linkId")
	sourceID := c.Param("sourceId")

	if err := h.linkService.DeleteSource(c.Request.Context(), userID, linkID, sourceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "source deleted"})
}
