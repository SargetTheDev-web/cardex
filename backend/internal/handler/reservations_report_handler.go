// internal/handler/reservations_report_handler.go

package handler

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReservationsReportCategory struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Count      int64   `json:"count"`
}

type ReservationsReportResponse struct {
	Status int `json:"status"`

	Meta struct {
		ReportType  string `json:"report_type"`
		ReportTitle string `json:"report_title"`
		Status      string `json:"status"`
		Filters     struct {
			DateRangeLabel string `json:"date_range_label"`
			FromDate       string `json:"from_date"`
			ToDate         string `json:"to_date"`
			Category       string `json:"category"`
			Role           string `json:"role"`
		} `json:"filters"`
	} `json:"meta"`

	Summary struct {
		Requests  int64 `json:"requests"`
		Pending   int64 `json:"pending"`
		Ready     int64 `json:"ready"`
		Fulfilled int64 `json:"fulfilled"`
	} `json:"summary"`

	CategoryBreakdown struct {
		TotalRecords int64                        `json:"total_records"`
		Items        []ReservationsReportCategory `json:"items"`
	} `json:"category_breakdown"`

	Transactions []repository.ReservationReportRow `json:"transactions"`
}

func GetReservationsReportHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromDate, err := time.Parse("2006-01-02", c.Query("from_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "from_date must use YYYY-MM-DD format",
			})
			return
		}

		toDate, err := time.Parse("2006-01-02", c.Query("to_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "to_date must use YYYY-MM-DD format",
			})
			return
		}

		if toDate.Before(fromDate) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "to_date must be on or after from_date",
			})
			return
		}

		var categoryID *int
		if value := c.Query("category_id"); value != "" {
			id, parseErr := strconv.Atoi(value)
			if parseErr != nil || id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": "category_id must be a positive integer",
				})
				return
			}
			categoryID = &id
		}

		var roleID *int
		if value := c.Query("role_id"); value != "" {
			id, parseErr := strconv.Atoi(value)
			if parseErr != nil || id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": "role_id must be a positive integer",
				})
				return
			}
			roleID = &id
		}

		rows, err := repository.GetReservationsReportRows(
			db,
			fromDate,
			toDate,
			categoryID,
			roleID,
		)
		if err != nil {
			log.Printf("GetReservationsReportRows failed: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to generate reservations report",
			})
			return
		}

		response := ReservationsReportResponse{
			Status:       http.StatusOK,
			Transactions: rows,
		}

		response.Meta.ReportType = "reservations_report"
		response.Meta.ReportTitle = "Reservations report"
		response.Meta.Status = "ready"
		response.Meta.Filters.DateRangeLabel = "Custom range"
		response.Meta.Filters.FromDate = fromDate.Format("2006-01-02")
		response.Meta.Filters.ToDate = toDate.Format("2006-01-02")
		response.Meta.Filters.Category = "All categories"
		response.Meta.Filters.Role = "All users"

		if categoryID != nil {
			response.Meta.Filters.Category = "Category ID: " + strconv.Itoa(*categoryID)
		}
		if roleID != nil {
			response.Meta.Filters.Role = "Role ID: " + strconv.Itoa(*roleID)
		}

		categoryCounts := make(map[string]int64)

		for _, row := range rows {
			response.Summary.Requests++
			categoryCounts[row.Category]++

			switch row.Status {
			case "Pending":
				response.Summary.Pending++
			case "Ready":
				response.Summary.Ready++
			case "Fulfilled":
				response.Summary.Fulfilled++
			}
		}

		response.CategoryBreakdown.Items = make(
			[]ReservationsReportCategory,
			0,
			len(categoryCounts),
		)

		for name, count := range categoryCounts {
			response.CategoryBreakdown.Items = append(
				response.CategoryBreakdown.Items,
				ReservationsReportCategory{
					Name:  name,
					Count: count,
				},
			)
			response.CategoryBreakdown.TotalRecords += count
		}

		for i := range response.CategoryBreakdown.Items {
			if response.CategoryBreakdown.TotalRecords > 0 {
				response.CategoryBreakdown.Items[i].Percentage =
					float64(response.CategoryBreakdown.Items[i].Count) * 100 /
						float64(response.CategoryBreakdown.TotalRecords)
			}
		}

		sort.Slice(response.CategoryBreakdown.Items, func(i, j int) bool {
			return response.CategoryBreakdown.Items[i].Name <
				response.CategoryBreakdown.Items[j].Name
		})

		c.JSON(http.StatusOK, response)
	}
}
