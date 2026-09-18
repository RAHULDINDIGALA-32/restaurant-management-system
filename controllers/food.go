package controller

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

// var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")
var validate = validator.New()

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

type FoodPaginationResult struct {
	Foods []models.Food `bson:"foods" json:"foods"`
	Total int64         `bson:"total" json:"total"`
}

func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Parse and validate pagination parameters.
		page, err := strconv.Atoi(c.DefaultQuery(
			"page",
			strconv.Itoa(defaultPage),
		))
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "page must be a positive integer",
			})
			return
		}

		limit, err := strconv.Atoi(c.DefaultQuery(
			"limit",
			strconv.Itoa(defaultPageSize),
		))
		if err != nil || limit < 1 || limit > maxPageSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "limit must be between 1 and 100",
			})
			return
		}

		// 2. Calculate the offset safely.
		skip := int64(page-1) * int64(limit)

		// 3. Create a request-scoped timeout.
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		// 4. Define the aggregation pipeline.
		pipeline := mongo.Pipeline{
			// Stable ordering for pagination.
			{{Key: "$sort", Value: bson.D{
				{Key: "_id", Value: 1},
			}}},

			// Get page data and total count together.
			{{Key: "$facet", Value: bson.D{
				{
					Key: "foods",
					Value: mongo.Pipeline{
						{{Key: "$skip", Value: skip}},
						{{Key: "$limit", Value: int64(limit)}},
					},
				},
				{
					Key: "metadata",
					Value: mongo.Pipeline{
						{{Key: "$count", Value: "total"}},
					},
				},
			}}},

			// Flatten metadata into a simple total field.
			{{Key: "$project", Value: bson.D{
				{
					Key:   "foods",
					Value: 1,
				},
				{
					Key: "total",
					Value: bson.D{
						{
							Key: "$ifNull",
							Value: bson.A{
								bson.D{
									{
										Key: "$arrayElemAt",
										Value: bson.A{
											"$metadata.total",
											0,
										},
									},
								},
								int64(0),
							},
						},
					},
				},
			}}},
		}

		// 5. Execute aggregation.
		cursor, err := foodCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch food items.",
			})
			return
		}
		defer cursor.Close(ctx)

		// 6. Decode the single aggregation result.
		result := FoodPaginationResult{
			Foods: []models.Food{},
			Total: 0,
		}

		if cursor.Next(ctx) {
			if err := cursor.Decode(&result); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to decode food items.",
				})
				return
			}
		} else if err := cursor.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read food items.",
			})
			return
		}

		// 7. Calculate pagination metadata.
		totalPages := (result.Total + int64(limit) - 1) /
			int64(limit)

		// 8. Return the paginated response.
		c.JSON(http.StatusOK, gin.H{
			"data": result.Foods,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"totalItems":  result.Total,
				"totalPages":  totalPages,
				"hasNextPage": int64(page) < totalPages,
				"hasPrevPage": page > 1,
			},
		})
	}
}

func GetFood() gin.HandlerFunc {
	return func(c *gin.Context) {

		foodID := c.Param("food_id")

		if foodID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "food_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var food models.Food

		err := foodCollection.
			FindOne(
				ctx,
				bson.M{"food_id": foodID},
			).
			Decode(&food)

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

		c.JSON(http.StatusOK, food)
	}
}

func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {

		var food models.Food
		var menu models.Menu

		if err := c.ShouldBindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		validatorErr := validate.Struct(food)
		if validatorErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": validatorErr.Error(),
			})
			return
		}

		if food.Price == nil ||
			math.IsNaN(*food.Price) ||
			math.IsInf(*food.Price, 0) ||
			*food.Price < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Price must be a valid, non-negative number",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		err := menuCollection.FindOne(
			ctx,
			bson.M{
				"menu_id": food.Menu_id,
			},
		).Decode(&menu)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Menu not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify menu",
			})
			return
		}

		currentTime := time.Now().UTC()

		food.ID = primitive.NewObjectID()
		food.Food_id = food.ID.Hex()
		food.Created_at = currentTime
		food.Updated_at = currentTime

		price := math.Round(*food.Price*100) / 100
		food.Price = &price

		_, insertErr := foodCollection.InsertOne(ctx, food)
		if insertErr != nil {
			msg := fmt.Sprintf("Failed to create food item.")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": msg,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Food item created successfully",
			"data":    food,
		})
	}
}

func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {

		foodID := c.Param("food_id")

		if foodID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Food ID is required",
			})
			return
		}

		var food models.Food

		if err := c.ShouldBindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		// Validate price if supplied.
		if food.Price != nil {
			if math.IsNaN(*food.Price) ||
				math.IsInf(*food.Price, 0) ||
				*food.Price < 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Price must be a valid, non-negative number",
				})
				return
			}

			price := math.Round(*food.Price*100) / 100
			food.Price = &price
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		// Build the update document.
		updateFields := bson.M{}

		if food.Name != nil {
			updateFields["name"] = *food.Name
		}

		if food.Price != nil {
			updateFields["price"] = *food.Price
		}

		if food.Image != nil {
			updateFields["image"] = *food.Image
		}

		// If menu_id is supplied, verify the menu exists.
		if food.Menu_id != nil {
			var menu models.Menu

			err := menuCollection.FindOne(
				ctx,
				bson.M{"menu_id": food.Menu_id},
			).Decode(&menu)

			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "Menu not found",
					})
					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to verify menu",
				})
				return
			}

			updateFields["menu_id"] = *food.Menu_id
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
			"food_id": foodID,
		}

		update := bson.M{
			"$set": updateFields,
		}

		result, err := foodCollection.UpdateOne(
			ctx,
			filter,
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update food item",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Food item not found",
			})
			return
		}

		// Fetch the updated food item.
		var updatedFood models.Food

		err = foodCollection.FindOne(
			ctx,
			filter,
		).Decode(&updatedFood)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Food updated, but failed to fetch updated data",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Food item updated successfully",
			"data":    updatedFood,
		})
	}
}

func DeleteFood() gin.HandlerFunc {
	return func(c *gin.Context) {

		foodID := c.Param("food_id")

		if foodID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Food ID is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		filter := bson.M{
			"food_id": foodID,
		}

		result, err := foodCollection.DeleteOne(
			ctx,
			filter,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete food item",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Food item not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Food item deleted successfully",
			"food_id": foodID,
		})
	}
}

/* Prod-Grade Arch

### Repo Layer

type FoodRepository interface {
    GetByID(ctx context.Context, foodID string) (*models.Food, error)
}

type foodRepository struct {
    collection *mongo.Collection
}

func (r *foodRepository) GetByID(
    ctx context.Context,
    foodID string,
) (*models.Food, error) {

    var food models.Food

    err := r.collection.
        FindOne(
            ctx,
            bson.M{"food_id": foodID},
        ).
        Decode(&food)

    if err != nil {
        return nil, err
    }

    return &food, nil
}


### Service Layer

type FoodService interface {
    GetFood(ctx context.Context, foodID string) (*models.Food, error)
}

type foodService struct {
    repo FoodRepository
}

func (s *foodService) GetFood(
    ctx context.Context,
    foodID string,
) (*models.Food, error) {

    return s.repo.GetByID(ctx, foodID)
}


### Controller Layer

type FoodController struct {
    service FoodService
}

func (fc *FoodController) GetFood(c *gin.Context) {

    foodID := c.Param("food_id")

    if foodID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "food_id is required",
        })
        return
    }

    ctx, cancel := context.WithTimeout(
        c.Request.Context(),
        5*time.Second,
    )
    defer cancel()

    food, err := fc.service.GetFood(ctx, foodID)

    if err != nil {

        if errors.Is(err, mongo.ErrNoDocuments) {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "food item not found",
            })
            return
        }

        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "failed to fetch food item",
        })
        return
    }

    c.JSON(http.StatusOK, food)
}

*/
