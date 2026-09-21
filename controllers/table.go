package controller

import (
	"context"
	"errors"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var tableCollection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		cursor, err := tableCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch tables",
			})
			return
		}
		defer cursor.Close(ctx)

		tables := make([]models.Table, 0)

		if err := cursor.All(ctx, &tables); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode tables",
			})
			return
		}

		c.JSON(http.StatusOK, tables)
	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("table_id")

		if tableID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "table_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var table models.Table

		err := tableCollection.FindOne(
			ctx,
			bson.M{"table_id": tableID},
		).Decode(&table)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Table not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch table",
			})
			return
		}

		c.JSON(http.StatusOK, table)
	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var table models.Table

		if err := c.ShouldBindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid table request body",
			})
			return
		}

		// Validate required fields.
		if table.Number_of_guests == nil ||
			table.Table_number == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "number_of_guests and table_number are required",
			})
			return
		}

		if *table.Number_of_guests < 1 ||
			*table.Table_number < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "number_of_guests and table_number must be positive",
			})
			return
		}

		now := time.Now().UTC()

		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()
		table.Created_at = now
		table.Updated_at = now

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		_, err := tableCollection.InsertOne(ctx, table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create table",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Table created successfully",
			"data":    table,
		})
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("table_id")

		if tableID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "table_id is required",
			})
			return
		}

		var request struct {
			Number_of_guests *int `json:"number_of_guests"`
			Table_number     *int `json:"table_number"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid table request body",
			})
			return
		}

		updateFields := bson.M{}

		if request.Number_of_guests != nil {
			if *request.Number_of_guests < 1 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "number_of_guests must be positive",
				})
				return
			}

			updateFields["number_of_guests"] =
				*request.Number_of_guests
		}

		if request.Table_number != nil {
			if *request.Table_number < 1 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "table_number must be positive",
				})
				return
			}

			updateFields["table_number"] =
				*request.Table_number
		}

		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one field must be provided",
			})
			return
		}

		updateFields["updated_at"] = time.Now().UTC()

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		result, err := tableCollection.UpdateOne(
			ctx,
			bson.M{"table_id": tableID},
			bson.M{"$set": updateFields},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update table",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Table not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Table updated successfully",
			"table_id": tableID,
		})
	}
}

func DeleteTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("table_id")

		if tableID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "table_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		result, err := tableCollection.DeleteOne(
			ctx,
			bson.M{"table_id": tableID},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete table",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Table not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Table deleted successfully",
			"table_id": tableID,
		})
	}
}
