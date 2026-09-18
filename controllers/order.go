package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")
var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		cursor, err := orderCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch Orders",
			})
			return
		}
		defer cursor.Close(ctx)

		var orders []models.Order

		if err := cursor.All(ctx, &orders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode orders",
			})
			return
		}

		if orders != nil {
			orders = []models.Order{}
		}

		c.JSON(http.StatusOK, orders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("order_id")

		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "order_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var order models.Order

		err := orderCollection.
			FindOne(
				ctx,
				bson.M{"order_id": orderID},
			).
			Decode(&order)

		if err != nil {

			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Food item Not Found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch foof item",
			})
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var order models.Order
		var table models.Table

		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		validatorErr := validate.Struct(order)
		if validatorErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": validatorErr.Error(),
			})
			return
		}

		if order.Order_date != nil {
			now := time.Now().UTC()

			orderDate := order.Order_date

			today := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				0, 0, 0, 0,
				time.UTC,
			)

			requestDate := time.Date(
				orderDate.Year(),
				orderDate.Month(),
				orderDate.Day(),
				0, 0, 0, 0,
				time.UTC,
			)

			if requestDate.Before(today) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "order_date should be today or future",
				})
				return
			}
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		err := tableCollection.FindOne(
			ctx,
			bson.M{
				"table_id": table.Table_id,
			},
		).Decode(&table)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Table not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify table",
			})
			return
		}

		currentTime := time.Now().UTC()

		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()
		order.Created_at = currentTime
		order.Updated_at = currentTime

		_, insertErr := foodCollection.InsertOne(ctx, order)
		if insertErr != nil {
			msg := fmt.Sprintf("Failed to create order.")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": msg,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Order created successfully",
			"data":    order,
		})
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("order_id")

		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "order ID is required",
			})
			return
		}

		var order models.Order

		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		// Build the update document.
		updateFields := bson.M{}

		if order.Order_date != nil {
			now := time.Now().UTC()

			orderDate := order.Order_date.UTC()

			today := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				0, 0, 0, 0,
				time.UTC,
			)

			requestedDate := time.Date(
				orderDate.Year(),
				orderDate.Month(),
				orderDate.Day(),
				0, 0, 0, 0,
				time.UTC,
			)

			if requestedDate.Before(today) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Order date must be today or future",
				})
				return
			}

			updateFields["order_date"] = *order.Order_date

		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		// If table_id is supplied, verify the table exists.
		if order.Table_id != nil {
			var table models.Table

			err := tableCollection.FindOne(
				ctx,
				bson.M{"table_id": order.Table_id},
			).Decode(&table)

			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "Table not found",
					})
					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to verify table",
				})
				return
			}

			updateFields["table_id"] = *order.Table_id
		}

		// Reject empty updates.
		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one field must be provided",
			})
			return
		}

		// Update timestamp.
		updateFields["updated_at"] = time.Now().UTC()

		filter := bson.M{
			"order_id": orderID,
		}

		update := bson.M{
			"$set": updateFields,
		}

		result, err := tableCollection.UpdateOne(
			ctx,
			filter,
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update order",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		// Fetch the updated order
		var updatedOrder models.Order

		err = orderCollection.FindOne(
			ctx,
			filter,
		).Decode(&updatedOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Order updated, but failed to fetch updated data",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Order updated successfully",
			"data":    updatedOrder,
		})
	}
}

func DeleteOrder() gin.HandlerFunc {
	return func(c *gin.Context) {

		orderID := c.Param("order_id")

		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "order_id id required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"order_id": orderID,
		}

		result, err := orderCollection.DeleteOne(
			ctx,
			filter,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete order",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order Not Found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Order deleted successfully",
			"Order ID": orderID,
		})

	}
}

func OrderItemOrderCreator(order models.Order) string {

	currentTime := time.Now().UTC()

	order.Created_at = currentTime
	order.Updated_at = currentTime
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orderCollection.InsertOne(ctx, order)

	return order.Order_id
}
