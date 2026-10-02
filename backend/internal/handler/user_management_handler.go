// internal/handler/user_management_handler.go

package handler

import (
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateUserRequest struct {
	Username        string  `json:"username" binding:"required"`
	EmailAddress    string  `json:"email_address" binding:"required,email"`
	Password        string  `json:"password" binding:"required,min=8"`
	RoleID          int     `json:"role_id" binding:"required"`
	InstitutionalID string  `json:"institutional_id" binding:"required"`
	LastName        string  `json:"last_name" binding:"required"`
	FirstName       string  `json:"first_name" binding:"required"`
	MiddleName      *string `json:"middle_name"`
	SuffixExtension *string `json:"suffix_extension"`
	Course          *string `json:"course"`
	MobileNumber    *string `json:"mobile_number"`
	BirthDate       *string `json:"birth_date"`
}

type UpdateUserRequest struct {
	Username        *string `json:"username"`
	EmailAddress    *string `json:"email_address"`
	InstitutionalID *string `json:"institutional_id"`
	LastName        *string `json:"last_name"`
	FirstName       *string `json:"first_name"`
	MiddleName      *string `json:"middle_name"`
	SuffixExtension *string `json:"suffix_extension"`
	Course          *string `json:"course"`
	MobileNumber    *string `json:"mobile_number"`
	BirthDate       *string `json:"birth_date"`
}

type ChangeUserStatusRequest struct {
	StatusID int `json:"status_id" binding:"required"`
}

func CreateUserHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateUserRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		adminIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "unauthorized",
			})
			return
		}

		adminID, ok := adminIDValue.(int)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "invalid user identity",
			})
			return
		}

		err := service.CreateUser(
			db,
			adminID,
			service.CreateUserInput{
				Username:        req.Username,
				EmailAddress:    req.EmailAddress,
				Password:        req.Password,
				RoleID:          req.RoleID,
				InstitutionalID: req.InstitutionalID,
				LastName:        req.LastName,
				FirstName:       req.FirstName,
				MiddleName:      req.MiddleName,
				SuffixExtension: req.SuffixExtension,
				Course:          req.Course,
				MobileNumber:    req.MobileNumber,
				BirthDate:       req.BirthDate,
			},
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": "user created successfully",
		})
	}
}

func GetUsersHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := service.GetUsers(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "failed to retrieve users",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   users,
		})
	}
}

func UpdateUserHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "invalid user ID",
			})
			return
		}

		var req UpdateUserRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		adminIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "unauthorized",
			})
			return
		}

		adminID, ok := adminIDValue.(int)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "invalid user identity",
			})
			return
		}

		err = service.UpdateUser(
			db,
			adminID,
			userID,
			service.UpdateUserInput{
				Username:        req.Username,
				EmailAddress:    req.EmailAddress,
				InstitutionalID: req.InstitutionalID,
				LastName:        req.LastName,
				FirstName:       req.FirstName,
				MiddleName:      req.MiddleName,
				SuffixExtension: req.SuffixExtension,
				Course:          req.Course,
				MobileNumber:    req.MobileNumber,
				BirthDate:       req.BirthDate,
			},
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "user updated successfully",
		})
	}
}

func ChangeUserStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "invalid user ID",
			})
			return
		}

		var req ChangeUserStatusRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		adminIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "unauthorized",
			})
			return
		}

		adminID, ok := adminIDValue.(int)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "invalid user identity",
			})
			return
		}

		err = service.ChangeUserStatus(
			db,
			adminID,
			userID,
			req.StatusID,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "user status updated successfully",
		})
	}
}
