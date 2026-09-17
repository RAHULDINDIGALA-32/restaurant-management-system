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

var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")

func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var menus []models.Menu

		cursor, err := menuCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch menus",
			})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode menus",
			})
			return
		}

		if menus != nil {
			menus = []models.Menu{}
		}

		c.JSON(http.StatusOK, menus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

		menuID := c.Param("menu_id")

		if menuID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "menu_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var menu models.Menu

		err := menuCollection.FindOne(
			ctx,
			bson.M{"menu_id": menuID},
		).Decode(&menu)

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

		c.JSON(http.StatusOK, menu)
	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

		var menu models.Menu

		if err := c.ShouldBindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		validatorErr := validate.Struct(menu)
		if validatorErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": validatorErr.Error(),
			})
			return
		}

		if menu.Start_date != nil &&
			menu.End_date != nil &&
			menu.End_date.Before(*menu.Start_date) {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "End date cannot be before start date",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		currentTime := time.Now().UTC()

		menu.ID = primitive.NewObjectID()
		menu.Menu_id = menu.ID.Hex()
		menu.Created_at = currentTime
		menu.Updated_at = currentTime

		_, insertErr := menuCollection.InsertOne(ctx, menu)

		if insertErr != nil {
			msg := fmt.Sprintf("Failed to create menu.")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": msg,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Menu created successfully",
			"data":    menu,
		})
	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

		menuID := c.Param("menu_id")

		if menuID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "menu_id is required",
			})
			return
		}

		var menu models.Menu

		if err := c.ShouldBindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		// Build update fields.
		updateFields := bson.M{}

		if menu.Name != "" {
			if err := validate.Var(
				menu.Name,
				"required,min=2,max=100",
			); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid menu name",
				})
				return
			}

			updateFields["name"] = menu.Name
		}

		if menu.Category != "" {
			if err := validate.Var(
				menu.Category,
				"required",
			); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid menu category",
				})
				return
			}

			updateFields["category"] = menu.Category
		}

		if menu.Start_date != nil {
			updateFields["start_date"] = *menu.Start_date
		}

		if menu.End_date != nil {
			updateFields["end_date"] = *menu.End_date
		}

		// Validate date range if both are provided.
		if menu.Start_date != nil &&
			menu.End_date != nil &&
			menu.End_date.Before(*menu.Start_date) {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "End date cannot be before start date",
			})
			return
		}

		// Reject empty updates.
		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one field must be provided",
			})
			return
		}

		// Set updated timestamp.
		updateFields["updated_at"] = time.Now().UTC()

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		filter := bson.M{
			"menu_id": menuID,
		}

		update := bson.M{
			"$set": updateFields,
		}

		result, err := menuCollection.UpdateOne(
			ctx,
			filter,
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update menu",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Menu not found",
			})
			return
		}

		// Fetch updated menu.
		var updatedMenu models.Menu

		err = menuCollection.FindOne(
			ctx,
			filter,
		).Decode(&updatedMenu)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Menu updated, but failed to fetch updated data",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Menu updated successfully",
			"data":    updatedMenu,
		})
	}
}

func DeleteMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

		menuID := c.Param("menu_id")

		if menuID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "menu_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		filter := bson.M{
			"menu_id": menuID,
		}

		result, err := menuCollection.DeleteOne(
			ctx,
			filter,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete menu",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Menu not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Menu deleted successfully",
			"menu_id": menuID,
		})
	}
}
