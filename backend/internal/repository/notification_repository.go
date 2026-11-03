// internal/repository/notification_repository.go

package repository

import (
	"errors"

	model "backend/internal/models"

	"gorm.io/gorm"
)

const (
	NotificationChannelInApp = "IN_APP"
	NotificationChannelEmail = "EMAIL"
)

// CreateNotification inserts a notification using the supplied database
// connection. It can be called with either a DB handle or a transaction.
func CreateNotification(
	tx *gorm.DB,
	notification *model.Notification,
) error {
	if notification == nil {
		return errors.New("notification is required")
	}

	return tx.Create(notification).Error
}

// GetUserNotifications returns only the authenticated user's in-app
// notifications, newest first.
func GetUserNotifications(
	db *gorm.DB,
	userID int,
	limit int,
	offset int,
) ([]model.Notification, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	if offset < 0 {
		offset = 0
	}

	notifications := make([]model.Notification, 0)

	err := db.
		Where(
			"user_id = ? AND channel = ?",
			userID,
			NotificationChannelInApp,
		).
		Order("created_at DESC, notification_id DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	if err != nil {
		return nil, err
	}

	return notifications, nil
}

// CountUnreadNotifications returns the user's unread in-app notification count.
func CountUnreadNotifications(
	db *gorm.DB,
	userID int,
) (int64, error) {
	var count int64

	err := db.
		Model(&model.Notification{}).
		Where(
			"user_id = ? AND channel = ? AND is_read = FALSE",
			userID,
			NotificationChannelInApp,
		).
		Count(&count).Error

	return count, err
}

// MarkNotificationAsRead updates a notification only if it belongs to
// the specified user.
func MarkNotificationAsRead(
	db *gorm.DB,
	userID int,
	notificationID int64,
) (bool, error) {
	result := db.
		Model(&model.Notification{}).
		Where(
			"notification_id = ? AND user_id = ? AND channel = ?",
			notificationID,
			userID,
			NotificationChannelInApp,
		).
		Update("is_read", true)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// MarkAllNotificationsAsRead marks all of the user's in-app notifications
// as read.
func MarkAllNotificationsAsRead(
	db *gorm.DB,
	userID int,
) (int64, error) {
	result := db.
		Model(&model.Notification{}).
		Where(
			"user_id = ? AND channel = ? AND is_read = FALSE",
			userID,
			NotificationChannelInApp,
		).
		Update("is_read", true)

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// DeleteNotification deletes only an in-app notification owned by the user.
func DeleteNotification(
	db *gorm.DB,
	userID int,
	notificationID int64,
) (bool, error) {
	result := db.
		Where(
			"notification_id = ? AND user_id = ? AND channel = ?",
			notificationID,
			userID,
			NotificationChannelInApp,
		).
		Delete(&model.Notification{})

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}
