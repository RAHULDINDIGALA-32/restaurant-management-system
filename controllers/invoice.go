package controller

import (
	"context"
	"errors"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type InvoiceViewFormat struct {
	Invoice_Id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		invoices := make([]models.Invoice, 0)

		cursor, err := invoiceCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch invoices",
			})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &invoices); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode invoices",
			})
			return
		}

		c.JSON(http.StatusOK, invoices)

	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		invoiceID := strings.TrimSpace(c.Param("invoice_id"))

		if invoiceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invoice_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var invoice models.Invoice

		err := invoiceCollection.FindOne(
			ctx,
			bson.M{"invoice_id": invoiceID},
		).Decode(&invoice)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Invoice not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to find invoice",
			})
			return
		}

		allOrderItems, err := ItemsByOrder(invoice.Order_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch order details",
			})
			return
		}

		if len(allOrderItems) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order details not found",
			})
			return
		}

		paymentMethod := "null"
		if invoice.Payment_method != nil {
			paymentMethod = *invoice.Payment_method
		}

		invoiceView := InvoiceViewFormat{
			Invoice_Id:       invoice.Invoice_id,
			Payment_method:   paymentMethod,
			Order_id:         invoice.Order_id,
			Payment_status:   invoice.Payment_status,
			Payment_due_date: invoice.Payment_due_date,
			Payment_due:      allOrderItems[0]["payment_due"],
			Table_number:     allOrderItems[0]["table_number"],
			Order_details:    allOrderItems[0]["order_details"],
		}

		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var invoice models.Invoice

		if err := c.ShouldBindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid invoice request body",
			})
			return
		}

		// Validate using the model's `validate` tags.
		validate := validator.New()
		validate.SetTagName("validate")

		if err := validate.Struct(invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		invoice.Invoice_id = strings.TrimSpace(invoice.Invoice_id)
		invoice.Order_id = strings.TrimSpace(invoice.Order_id)

		if invoice.Invoice_id == "" || invoice.Order_id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invoice_id and order_id cannot be empty",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		// Check whether invoice ID already exists.
		count, err := invoiceCollection.CountDocuments(
			ctx,
			bson.M{"invoice_id": invoice.Invoice_id},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check invoice ID",
			})
			return
		}

		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Invoice ID already exists",
			})
			return
		}

		now := time.Now().UTC()

		invoice.ID = primitive.NewObjectID()
		invoice.Created_at = now
		invoice.Updated_at = now

		if invoice.Payment_due_date.IsZero() {
			invoice.Payment_due_date = now
		}

		_, err := invoiceCollection.InsertOne(ctx, invoice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create invoice",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Invoice created successfully",
			"data":    invoice,
		})
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		invoiceID := strings.TrimSpace(c.Param("invoice_id"))

		if invoiceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invoice_id is required",
			})
			return
		}

		// Pointers allow omitted fields to remain unchanged.
		var request struct {
			Payment_method   *string    `json:"payment_method"`
			Payment_status   *string    `json:"payment_status"`
			Payment_due_date *time.Time `json:"payment_due_date"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid update request body",
			})
			return
		}

		updates := bson.M{}

		if request.Payment_method != nil {
			if *request.Payment_method != "CARD" &&
				*request.Payment_method != "CASH" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "payment_method must be CARD or CASH",
				})
				return
			}

			updates["payment_method"] = *request.Payment_method
		}

		if request.Payment_status != nil {
			if *request.Payment_status != "PENDING" &&
				*request.Payment_status != "PAID" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "payment_status must be PENDING or PAID",
				})
				return
			}

			updates["payment_status"] = *request.Payment_status
		}

		if request.Payment_due_date != nil {
			if request.Payment_due_date.IsZero() {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "payment_due_date cannot be zero",
				})
				return
			}

			updates["payment_due_date"] = *request.Payment_due_date
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No valid fields provided for update",
			})
			return
		}

		updates["updated_at"] = time.Now().UTC()

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		result, err := invoiceCollection.UpdateOne(
			ctx,
			bson.M{"invoice_id": invoiceID},
			bson.M{"$set": updates},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update invoice",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Invoice not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Invoice updated successfully",
			"invoice_id": invoiceID,
		})
	}
}

func DeleteInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {

		invoiceID := c.Param("invoice_id")

		if invoiceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invoice_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		result, err := invoiceCollection.DeleteOne(
			ctx,
			bson.M{
				"invoice_id": invoiceID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete invoice",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Invoice Not Found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Invoice deleted successfully",
			"invoice_id": invoiceID,
		})
	}
}
