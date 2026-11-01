package handler

import (
	"net/http"
	"strconv"
	"time"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CirculationSummaryResponse struct {
	Status int `json:"status"`
	Meta   struct {
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
		TotalTransactions int64 `json:"total_transactions"`
		BooksBorrowed     int64 `json:"books_borrowed"`
		BooksReturned     int64 `json:"books_returned"`
		OverdueIncidents  int64 `json:"overdue_incidents"`
	} `json:"summary"`

	CategoryBreakdown struct {
		TotalRecords int64                                 `json:"total_records"`
		Items        []repository.CirculationCategoryCount `json:"items"`
	} `json:"category_breakdown"`

	Transactions []repository.CirculationReportRow `json:"transactions"`
}

func GetCirculationSummaryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromDate, err := time.Parse("2006-01-02", c.Query("from_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "from_date must use YYYY-MM-DD format",
			})
			return
		}

		toDate, err := time.Parse("2006-01-02", c.Query("to_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "to_date must use YYYY-MM-DD format",
			})
			return
		}

		if toDate.Before(fromDate) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "to_date must be on or after from_date",
			})
			return
		}

		var categoryID *int
		if value := c.Query("category_id"); value != "" {
			id, parseErr := strconv.Atoi(value)
			if parseErr != nil || id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
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
					"status":  "error",
					"message": "role_id must be a positive integer",
				})
				return
			}
			roleID = &id
		}

		rows, err := repository.GetCirculationReportRows(
			db,
			fromDate,
			toDate,
			categoryID,
			roleID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to generate circulation report",
			})
			return
		}

		response := CirculationSummaryResponse{}
		response.Meta.ReportType = "circulation_summary"
		response.Meta.ReportTitle = "Circulation summary"
		response.Meta.Status = "ready"
		response.Meta.Filters.FromDate = fromDate.Format("2006-01-02")
		response.Meta.Filters.ToDate = toDate.Format("2006-01-02")
		response.Meta.Filters.DateRangeLabel = "Custom range"
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
			response.Summary.TotalTransactions++

			switch row.Type {
			case "Borrow":
				response.Summary.BooksBorrowed++
			case "Return":
				response.Summary.BooksReturned++
				if row.Overdue {
					response.Summary.OverdueIncidents++
				}
			}

			categoryCounts[row.Category]++
		}

		response.CategoryBreakdown.Items = make(
			[]repository.CirculationCategoryCount,
			0,
			len(categoryCounts),
		)

		for name, count := range categoryCounts {
			response.CategoryBreakdown.Items = append(
				response.CategoryBreakdown.Items,
				repository.CirculationCategoryCount{
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

		response.Transactions = rows

		response.Status = http.StatusOK
		c.JSON(http.StatusOK, response)
	}
}
