package handler

import (
	"net/http"
	"strconv"

	"payment-platform/internal/notification/port"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	repo port.NotificationRepository
}

func New(repo port.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	notifs := rg.Group("/notifications")
	notifs.GET("", h.List)
	notifs.GET("/unread-count", h.UnreadCount)
	notifs.PATCH("/:id/read", h.MarkRead)
	notifs.POST("/read-all", h.MarkAllRead)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifs, total, err := h.repo.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifs,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	count, err := h.repo.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.repo.MarkRead(c.Request.Context(), c.Param("id"), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
