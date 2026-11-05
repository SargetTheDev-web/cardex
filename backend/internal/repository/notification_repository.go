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
) ([]model.NotificationResponse, error) {
	return getNotificationResponses(db, userID, limit, offset, false)
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

// GetUnreadUserNotifications returns the authenticated user's unread
// in-app notifications, newest first.
func GetUnreadUserNotifications(
	db *gorm.DB,
	userID int,
	limit int,
	offset int,
) ([]model.NotificationResponse, error) {
	return getNotificationResponses(db, userID, limit, offset, true)
}

func getNotificationResponses(
	db *gorm.DB,
	userID int,
	limit int,
	offset int,
	unreadOnly bool,
) ([]model.NotificationResponse, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notifications := make([]model.NotificationResponse, 0)

	query := db.Table("notification n").
		Select(`
			n.notification_id,
			n.user_id,
			TRIM(CONCAT_WS(' ',
				NULLIF(up.first_name, ''),
				NULLIF(up.middle_name, ''),
				NULLIF(up.last_name, ''),
				NULLIF(up.suffix_extension, '')
			)) AS name,
			ur.role_code AS role,
			up.institutional_id,
			n.type,
			n.channel,
			n.subject,
			n.message,
			n.is_read,
			n.created_at,
			n.reference_type,
			n.reference_id
		`).
		Joins(`JOIN "user" u ON u.user_id = n.user_id`).
		Joins(`LEFT JOIN user_profile up ON up.user_id = u.user_id`).
		Joins(`LEFT JOIN user_role ur ON ur.role_id = u.role_id`).
		Where("n.user_id = ? AND n.channel = ?", userID, NotificationChannelInApp)

	if unreadOnly {
		query = query.Where("n.is_read = FALSE")
	}

	err := query.
		Order("n.created_at DESC, n.notification_id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&notifications).Error
	if err != nil {
		return nil, err
	}

	return notifications, nil
}
