// nternal/service/rbac_service.go

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

type RolePermissionUpdate struct {
	RoleID        int   `json:"role_id"`
	PermissionIDs []int `json:"permission_ids"`
}

type RBACMatrixResponse struct {
	Roles       []model.UserRole                      `json:"roles"`
	Permissions []model.Permission                    `json:"permissions"`
	Assignments []repository.RolePermissionAssignment `json:"assignments"`
}

func GetRBACMatrix(db *gorm.DB) (*RBACMatrixResponse, error) {
	roles, err := repository.GetAllRoles(db)
	if err != nil {
		return nil, err
	}

	permissions, err := repository.GetAllPermissions(db)
	if err != nil {
		return nil, err
	}

	assignments, err := repository.GetRolePermissionMatrix(db)
	if err != nil {
		return nil, err
	}

	return &RBACMatrixResponse{
		Roles:       roles,
		Permissions: permissions,
		Assignments: assignments,
	}, nil
}

func UpdateRBACMatrix(
	db *gorm.DB,
	actorID int,
	updates []RolePermissionUpdate,
	ip string,
	userAgent string,
) error {
	if len(updates) != 6 {
		return errors.New("all six roles must be submitted")
	}

	desired := make(map[int]map[int]struct{}, len(updates))
	roleIDs := make([]int, 0, len(updates))
	permissionIDs := make([]int, 0)

	for _, update := range updates {
		if update.RoleID <= 0 {
			return errors.New("invalid role ID")
		}

		if _, duplicate := desired[update.RoleID]; duplicate {
			return fmt.Errorf("duplicate role ID: %d", update.RoleID)
		}

		desired[update.RoleID] = make(map[int]struct{})

		for _, permissionID := range update.PermissionIDs {
			if permissionID <= 0 {
				return fmt.Errorf("invalid permission ID: %d", permissionID)
			}

			if _, duplicate := desired[update.RoleID][permissionID]; duplicate {
				return fmt.Errorf(
					"duplicate permission ID %d for role %d",
					permissionID,
					update.RoleID,
				)
			}

			desired[update.RoleID][permissionID] = struct{}{}
			permissionIDs = append(permissionIDs, permissionID)
		}

		roleIDs = append(roleIDs, update.RoleID)
	}

	sort.Ints(roleIDs)

	if len(roleIDs) != 6 {
		return errors.New("invalid role submission")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Lock role records to prevent concurrent matrix modifications.
		var roles []model.UserRole

		if err := tx.
			Clauses().
			Set("gorm:query_option", "FOR UPDATE").
			Where("role_id IN ?", roleIDs).
			Order("role_id ASC").
			Find(&roles).Error; err != nil {
			return err
		}

		if len(roles) != 6 {
			return errors.New("all six configured roles must exist")
		}

		if err := repository.ValidateRolePermissionIDs(
			tx,
			roleIDs,
			permissionIDs,
		); err != nil {
			return err
		}

		// Capture the previous matrix for the audit record.
		oldAssignments, err := repository.GetRolePermissionMatrix(tx)
		if err != nil {
			return err
		}

		if err := repository.UpdateRolePermissionMatrix(
			tx,
			roles,
			desired,
			actorID,
		); err != nil {
			return err
		}

		newAssignments, err := repository.GetRolePermissionMatrix(tx)
		if err != nil {
			return err
		}

		changes := map[string]interface{}{
			"before": oldAssignments,
			"after":  newAssignments,
		}

		description, err := json.Marshal(changes)
		if err != nil {
			return err
		}

		actionID, err := repository.GetAuditActionID(tx, "UPDATE")
		if err != nil {
			return err
		}

		moduleID, err := repository.GetAuditModuleID(tx, "RBAC")
		if err != nil {
			return err
		}

		userID := actorID

		return repository.InsertAuditLog(
			tx,
			&userID,
			actionID,
			moduleID,
			"Updated RBAC permission matrix: "+string(description),
			ip,
			userAgent,
		)
	})
}
