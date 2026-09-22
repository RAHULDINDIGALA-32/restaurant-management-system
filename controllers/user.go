package controller

import (
	"context"
	"errors"
	"net/http"
	"restaurant-management-system/database"
	helper "restaurant-management-system/helpers"
	"restaurant-management-system/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var sessionCollection *mongo.Collection = database.OpenCollection(database.Client, "session")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{
			{"$match", bson.D{{}}},
		}

		projectStage := bson.D{
			{"$project", bson.D{
				{"_id", 0},
				{"user_items", bson.D{
					{"$slice", []interface{}{"$data", startIndex, recordPerPage}},
				}},
			}},
		}

		cursor, err := userCollection.Aggregate(
			ctx,
			mongo.Pipeline{
				matchStage,
				projectStage,
			},
		)
		defer cursor.Close(ctx)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch users",
			})
			return
		}

		var result []bson.M

		if err := cursor.All(ctx, &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode users",
			})
			return
		}

		c.JSON(http.StatusOK, result)

	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User Not Found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
			return
		}

		if err := validate.Struct(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}

		if user.Email == nil || user.Password == nil || user.FirstName == nil || user.LastName == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email, password, first_name and last_name are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		emailValue := strings.TrimSpace(strings.ToLower(*user.Email))
		emailCount, err := userCollection.CountDocuments(ctx, bson.M{"email": emailValue})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check email"})
			return
		}
		if emailCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already in use"})
			return
		}

		if user.Phone != nil {
			phoneCount, err := userCollection.CountDocuments(ctx, bson.M{"phone": *user.Phone})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check phone number"})
				return
			}
			if phoneCount > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
				return
			}
		}

		hashedPassword, err := helper.HashPassword(*user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = &hashedPassword
		user.Email = &emailValue

		user.ID = primitive.NewObjectID()
		user.UserID = user.ID.Hex()
		if user.Role == "" {
			user.Role = "user"
		}

		now := time.Now().UTC()
		user.CreatedAt = now
		user.UpdatedAt = now

		accessToken, refreshToken, err := helper.GenerateAllTokens(emailValue, *user.FirstName, *user.LastName, user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
			return
		}

		tokenHash, err := helper.HashPassword(refreshToken)
		if err == nil {
			sessionDoc := models.Session{
				ID:        primitive.NewObjectID(),
				UserID:    user.UserID,
				TokenHash: tokenHash,
				ExpiresAt: now.Add(7 * 24 * time.Hour),
				CreatedAt: now,
				UserAgent: c.Request.UserAgent(),
				IPAddress: c.ClientIP(),
			}
			_, _ = sessionCollection.InsertOne(ctx, sessionDoc)
		}

		result, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "User registered successfully",
			"user":          user,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"inserted_id":   result.InsertedID,
		})
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
			return
		}

		request.Email = strings.TrimSpace(strings.ToLower(request.Email))
		if request.Email == "" || request.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"email": request.Email}).Decode(&user)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		if user.Password == nil || !helper.VerifyPassword(*user.Password, request.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		accessToken, refreshToken, err := helper.GenerateAllTokens(request.Email, *user.FirstName, *user.LastName, user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
			return
		}

		tokenHash, err := helper.HashPassword(refreshToken)
		if err == nil {
			sessionDoc := models.Session{
				ID:        primitive.NewObjectID(),
				UserID:    user.UserID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
				CreatedAt: time.Now().UTC(),
				UserAgent: c.Request.UserAgent(),
				IPAddress: c.ClientIP(),
			}
			_, _ = sessionCollection.InsertOne(ctx, sessionDoc)
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Login successful",
			"user":          user,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		var updateData struct {
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
			Phone     *string `json:"phone"`
			Avatar    *string `json:"avatar"`
			Role      *string `json:"role"`
		}

		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
			return
		}

		updateFields := bson.M{"updated_at": time.Now().UTC()}
		if updateData.FirstName != nil {
			updateFields["first_name"] = *updateData.FirstName
		}
		if updateData.LastName != nil {
			updateFields["last_name"] = *updateData.LastName
		}
		if updateData.Phone != nil {
			updateFields["phone"] = *updateData.Phone
		}
		if updateData.Avatar != nil {
			updateFields["avatar"] = *updateData.Avatar
		}
		if updateData.Role != nil {
			updateFields["role"] = *updateData.Role
		}

		if len(updateFields) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		result, err := userCollection.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": updateFields})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User Not Found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User updated successfully", "user_id": userID})
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		result, err := userCollection.DeleteOne(ctx, bson.M{"user_id": userID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}
		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User Not Found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully", "user_id": userID})
	}
}
