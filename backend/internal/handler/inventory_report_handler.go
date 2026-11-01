// internal/handler/inventory_report_handler.go

package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/repository"
)

type InventoryReportResponse struct {
	Status             int                         `json:"status"`
	Meta               InventoryReportMeta         `json:"meta"`
	Summary            InventoryReportSummary      `json:"summary"`
	HoldingsByCategory InventoryHoldingsByCategory `json:"holdings_by_category"`
	Transactions       []any                       `json:"transactions"`
}

type InventoryReportMeta struct {
	ReportType  string                 `json:"report_type"`
	ReportTitle string                 `json:"report_title"`
	Status      string                 `json:"status"`
	Filters     InventoryReportFilters `json:"filters"`
}

type InventoryReportFilters struct {
	DateRangeLabel string `json:"date_range_label"`
	FromDate       string `json:"from_date"`
	ToDate         string `json:"to_date"`
	Category       string `json:"category"`
	Role           string `json:"role"`
}

type InventoryReportSummary struct {
	TitlesRepresented int64 `json:"titles_represented"`
	Borrowed          int64 `json:"borrowed"`
	Returned          int64 `json:"returned"`
	OnHold            int64 `json:"on_hold"`
}

type InventoryHoldingsByCategory struct {
	TotalRecords int64                   `json:"total_records"`
	Items        []InventoryCategoryItem `json:"items"`
}

type InventoryCategoryItem struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Count      int64   `json:"count"`
}

func GetInventoryReportHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromDate, err := time.Parse("2006-01-02", c.Query("from_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": 400,
				"error":  "from_date must use YYYY-MM-DD format",
			})
			return
		}

		toDate, err := time.Parse("2006-01-02", c.Query("to_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": 400,
				"error":  "to_date must use YYYY-MM-DD format",
			})
			return
		}

		if toDate.Before(fromDate) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": 400,
				"error":  "to_date must be on or after from_date",
			})
			return
		}

		categoryID, err := parseOptionalPositiveInt(c.Query("category_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": 400,
				"error":  "category_id must be a positive integer",
			})
			return
		}

		roleID, err := parseOptionalPositiveInt(c.Query("role_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": 400,
				"error":  "role_id must be a positive integer",
			})
			return
		}

		rows, err := repository.GetInventoryReport(
			db,
			fromDate,
			toDate,
			categoryID,
			roleID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": 500,
				"error":  "failed to retrieve inventory report",
			})
			return
		}

		categories := make([]InventoryCategoryItem, 0, len(rows.Categories))
		var totalCopies int64

		for _, row := range rows.Categories {
			totalCopies += row.Count
		}

		for _, row := range rows.Categories {
			percentage := 0.0
			if totalCopies > 0 {
				percentage = float64(row.Count) / float64(totalCopies) * 100
			}

			categories = append(categories, InventoryCategoryItem{
				Name:       row.CategoryName,
				Percentage: percentage,
				Count:      row.Count,
			})
		}

		sort.Slice(categories, func(i, j int) bool {
			return categories[i].Name < categories[j].Name
		})

		categoryLabel := "All categories"
		if categoryID != nil {
			categoryLabel = "Category ID: " + strconv.Itoa(*categoryID)
		}

		roleLabel := "All users"
		if roleID != nil {
			roleLabel = "Role ID: " + strconv.Itoa(*roleID)
		}

		response := InventoryReportResponse{
			Status: 200,
			Meta: InventoryReportMeta{
				ReportType:  "inventory_report",
				ReportTitle: "Inventory report",
				Status:      "ready",
				Filters: InventoryReportFilters{
					DateRangeLabel: "Custom range",
					FromDate:       fromDate.Format("2006-01-02"),
					ToDate:         toDate.Format("2006-01-02"),
					Category:       categoryLabel,
					Role:           roleLabel,
				},
			},
			Summary: InventoryReportSummary{
				TitlesRepresented: rows.TitlesRepresented,
				Borrowed:          rows.Borrowed,
				Returned:          rows.Returned,
				OnHold:            rows.OnHold,
			},
			HoldingsByCategory: InventoryHoldingsByCategory{
				TotalRecords: totalCopies,
				Items:        categories,
			},
			Transactions: []any{},
		}

		c.JSON(http.StatusOK, response)
	}
}

func parseOptionalPositiveInt(value string) (*int, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return nil, strconv.ErrSyntax
	}

	return &parsed, nil
}
