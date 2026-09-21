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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var noteCollection = database.OpenCollection(database.Client, "note")

func GetNotes() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		cursor, err := noteCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch notes",
			})
			return
		}
		defer cursor.Close(ctx)

		notes := make([]models.Note, 0)

		if err := cursor.All(ctx, &notes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode notes",
			})
			return
		}

		c.JSON(http.StatusOK, notes)
	}
}

func GetNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		noteID := c.Param("note_id")

		if noteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "note_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var note models.Note

		err := noteCollection.FindOne(
			ctx,
			bson.M{"note_id": noteID},
		).Decode(&note)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Note not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch note",
			})
			return
		}

		c.JSON(http.StatusOK, note)
	}
}

func CreateNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		var note models.Note

		if err := c.ShouldBindJSON(&note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid note request body",
			})
			return
		}

		note.Title = strings.TrimSpace(note.Title)
		note.Text = strings.TrimSpace(note.Text)

		if note.Title == "" || note.Text == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Title and text are required",
			})
			return
		}

		now := time.Now().UTC()

		note.ID = primitive.NewObjectID()
		note.Note_id = note.ID.Hex()
		note.Created_at = now
		note.Updated_at = now

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		_, err := noteCollection.InsertOne(ctx, note)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create note",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Note created successfully",
			"data":    note,
		})
	}
}

func UpdateNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		noteID := c.Param("note_id")

		if noteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "note_id is required",
			})
			return
		}

		var request struct {
			Title *string `json:"title"`
			Text  *string `json:"text"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid note request body",
			})
			return
		}

		updateFields := bson.M{}

		if request.Title != nil {
			title := strings.TrimSpace(*request.Title)

			if title == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Title cannot be empty",
				})
				return
			}

			updateFields["title"] = title
		}

		if request.Text != nil {
			text := strings.TrimSpace(*request.Text)

			if text == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Text cannot be empty",
				})
				return
			}

			updateFields["text"] = text
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

		result, err := noteCollection.UpdateOne(
			ctx,
			bson.M{"note_id": noteID},
			bson.M{"$set": updateFields},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update note",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Note not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Note updated successfully",
			"note_id": noteID,
		})
	}
}

func DeleteNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		noteID := c.Param("note_id")

		if noteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "note_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		result, err := noteCollection.DeleteOne(
			ctx,
			bson.M{"note_id": noteID},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete note",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Note not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Note deleted successfully",
			"note_id": noteID,
		})
	}
}
