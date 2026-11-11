// internal/repository/role_repository.go

package repository

import (
	model "backend/internal/models"
	"fmt"

	"gorm.io/gorm"
)

func GetUserRole(db *gorm.DB, userID int) (*model.UserRole, error) {
	var role model.UserRole

	err := db.
		Table("user_role ur").
		Select(`
			ur.role_id,
			ur.role_code,
			ur.role_name,
			ur.is_active,
			ur.description,
			ur.created_by,
			ur.updated_by
		`).
		Joins(`JOIN "user" u ON u.role_id = ur.role_id`).
		Where("u.user_id = ?", userID).
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return &role, nil
}

func GetRolePermissions(
	db *gorm.DB,
	roleID int,
) ([]string, error) {

	var permissions []string

	err := db.
		Table("role_permission rp").
		Select("p.permission_code").
		Joins(
			"JOIN permission p ON p.permission_id = rp.permission_id",
		).
		Where("rp.role_id = ?", roleID).
		Order("p.permission_code ASC").
		Pluck("p.permission_code", &permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}

// UserHasPermission checks whether the user's current active role
// has the specified permission.
func UserHasPermission(
	db *gorm.DB,
	userID int,
	permissionCode string,
) (bool, error) {
	var allowed bool

	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM "user" u
			JOIN user_role ur
				ON ur.role_id = u.role_id
			JOIN role_permission rp
				ON rp.role_id = ur.role_id
			JOIN permission p
				ON p.permission_id = rp.permission_id
			WHERE u.user_id = ?
				AND ur.is_active = TRUE
				AND p.permission_code = ?
		)
	`, userID, permissionCode).Scan(&allowed).Error

	if err != nil {
		return false, err
	}

	return allowed, nil
}

type RolePermissionAssignment struct {
	RoleID       int `gorm:"column:role_id" json:"role_id"`
	PermissionID int `gorm:"column:permission_id" json:"permission_id"`
}

func GetAllRoles(db *gorm.DB) ([]model.UserRole, error) {
	var roles []model.UserRole

	err := db.
		Order("role_id ASC").
		Find(&roles).Error

	return roles, err
}

func GetAllPermissions(db *gorm.DB) ([]model.Permission, error) {
	var permissions []model.Permission

	err := db.
		Order("permission_id ASC").
		Find(&permissions).Error

	return permissions, err
}

func GetRolePermissionMatrix(
	db *gorm.DB,
) ([]RolePermissionAssignment, error) {
	var assignments []RolePermissionAssignment

	err := db.
		Table("role_permission").
		Select("role_id, permission_id").
		Order("role_id ASC, permission_id ASC").
		Scan(&assignments).Error

	return assignments, err
}

func UpdateRolePermissionMatrix(
	tx *gorm.DB,
	roles []model.UserRole,
	desired map[int]map[int]struct{},
	actorID int,
) error {
	for _, role := range roles {
		roleID := role.RoleID
		desiredPermissions := desired[roleID]

		var existing []model.RolePermission

		if err := tx.
			Clauses().
			Where("role_id = ?", roleID).
			Find(&existing).Error; err != nil {
			return err
		}

		existingSet := make(map[int]model.RolePermission, len(existing))

		for _, assignment := range existing {
			existingSet[assignment.PermissionID] = assignment
		}

		// Revoke permissions no longer selected.
		for permissionID, assignment := range existingSet {
			if _, keep := desiredPermissions[permissionID]; keep {
				continue
			}

			if err := tx.
				Where(
					"role_id = ? AND permission_id = ?",
					roleID,
					permissionID,
				).
				Delete(&model.RolePermission{}).Error; err != nil {
				return err
			}

			_ = assignment
		}

		// Assign newly selected permissions.
		for permissionID := range desiredPermissions {
			if _, exists := existingSet[permissionID]; exists {
				continue
			}

			assignment := model.RolePermission{
				RoleID:       roleID,
				PermissionID: permissionID,
				CreatedBy:    &actorID,
				UpdatedBy:    &actorID,
			}

			if err := tx.Create(&assignment).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateRolePermissionIDs(
	tx *gorm.DB,
	roleIDs []int,
	permissionIDs []int,
) error {
	uniqueRoleIDs := make(map[int]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		if id <= 0 {
			return fmt.Errorf("invalid role ID")
		}
		uniqueRoleIDs[id] = struct{}{}
	}

	uniquePermissionIDs := make(map[int]struct{}, len(permissionIDs))
	for _, id := range permissionIDs {
		if id <= 0 {
			return fmt.Errorf("invalid permission ID")
		}
		uniquePermissionIDs[id] = struct{}{}
	}

	if len(uniqueRoleIDs) != len(roleIDs) {
		return fmt.Errorf("duplicate role IDs are not allowed")
	}

	roleIDList := make([]int, 0, len(uniqueRoleIDs))
	for id := range uniqueRoleIDs {
		roleIDList = append(roleIDList, id)
	}

	var roleCount int64
	if len(roleIDList) > 0 {
		if err := tx.
			Table("user_role").
			Where("role_id IN ?", roleIDList).
			Count(&roleCount).Error; err != nil {
			return err
		}
	}

	if roleCount != int64(len(roleIDList)) {
		return fmt.Errorf("one or more role IDs do not exist")
	}

	permissionIDList := make([]int, 0, len(uniquePermissionIDs))
	for id := range uniquePermissionIDs {
		permissionIDList = append(permissionIDList, id)
	}

	var permissionCount int64
	if len(permissionIDList) > 0 {
		if err := tx.
			Table("permission").
			Where("permission_id IN ?", permissionIDList).
			Count(&permissionCount).Error; err != nil {
			return err
		}
	}

	if permissionCount != int64(len(permissionIDList)) {
		return fmt.Errorf("one or more permission IDs do not exist")
	}

	return nil
}
