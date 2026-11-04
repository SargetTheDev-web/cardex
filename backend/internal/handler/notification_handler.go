// internal/handler/notification_handler.go

package handler

import (
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func notificationUserID(c *gin.Context) (int, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": http.StatusUnauthorized,
			"error":  "unauthorized",
		})
		return 0, false
	}

	userID, ok := value.(int)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": http.StatusUnauthorized,
			"error":  "invalid user session",
		})
		return 0, false
	}

	return userID, true
}

func notificationPagination(c *gin.Context) (int, int, bool) {
	limit := 20
	offset := 0

	if rawLimit := c.Query("limit"); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value < 1 || value > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "limit must be between 1 and 100",
			})
			return 0, 0, false
		}
		limit = value
	}

	if rawOffset := c.Query("offset"); rawOffset != "" {
		value, err := strconv.Atoi(rawOffset)
		if err != nil || value < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "offset must be zero or greater",
			})
			return 0, 0, false
		}
		offset = value
	}

	return limit, offset, true
}

func GetNotificationsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		limit, offset, ok := notificationPagination(c)
		if !ok {
			return
		}

		svc := service.NewNotificationService(db, nil)
		notifications, err := svc.GetUserNotifications(userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve notifications",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        http.StatusOK,
			"notifications": notifications,
			"limit":         limit,
			"offset":        offset,
		})
	}
}

func GetUnreadNotificationsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		limit, offset, ok := notificationPagination(c)
		if !ok {
			return
		}

		svc := service.NewNotificationService(db, nil)
		notifications, err := svc.GetUnreadUserNotifications(userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve unread notifications",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        http.StatusOK,
			"notifications": notifications,
			"limit":         limit,
			"offset":        offset,
		})
	}
}

func GetUnreadNotificationCountHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		svc := service.NewNotificationService(db, nil)
		count, err := svc.GetUnreadCount(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve unread notification count",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       http.StatusOK,
			"unread_count": count,
		})
	}
}

func MarkNotificationAsReadHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || notificationID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid notification ID",
			})
			return
		}

		svc := service.NewNotificationService(db, nil)
		updated, err := svc.MarkAsRead(userID, notificationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to mark notification as read",
			})
			return
		}
		if !updated {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  "notification not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "notification marked as read",
		})
	}
}

func MarkAllNotificationsAsReadHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		svc := service.NewNotificationService(db, nil)
		updatedCount, err := svc.MarkAllAsRead(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to mark notifications as read",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        http.StatusOK,
			"message":       "notifications marked as read",
			"updated_count": updatedCount,
		})
	}
}

func DeleteNotificationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := notificationUserID(c)
		if !ok {
			return
		}

		notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || notificationID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid notification ID",
			})
			return
		}

		svc := service.NewNotificationService(db, nil)
		deleted, err := svc.Delete(userID, notificationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to delete notification",
			})
			return
		}
		if !deleted {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  "notification not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "notification deleted",
		})
	}
}
