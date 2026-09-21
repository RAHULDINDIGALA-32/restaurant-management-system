package controller

import (
	"context"
	"errors"
	"math"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		cursor, err := orderItemCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch order items",
			})
			return
		}
		defer cursor.Close(ctx)

		//orderItems := []models.OrderItem{}
		orderItems := make([]models.OrderItem, 0)

		if err := cursor.All(ctx, &orderItems); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode order items",
			})
			return
		}

		c.JSON(http.StatusOK, orderItems)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderItemID := c.Param("orderItem_id")

		if orderItemID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "orderItem_id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		var orderItem models.Menu

		err := orderItemCollection.FindOne(
			ctx,
			bson.M{"orderItem_id": orderItemID},
		).Decode(&orderItem)

		if err != nil {

			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Order item Not Found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch order item",
			})
			return
		}

		c.JSON(http.StatusOK, orderItem)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {

		orderID := c.Param("order_id")

		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Order ID is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		orderItems, err := ItemsByOrder(orderID, ctx)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrive items",
			})
			return
		}

		c.JSON(http.StatusOK, orderItems)

	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {

		var orderItemPack OrderItemPack

		if err := c.ShouldBindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid orderItems request body",
			})
			return
		}

		if len(orderItemPack.Order_items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one order item is required",
			})
			return
		}

		now := time.Now().UTC()
		order := models.Order{
			Order_date: &now,
			Table_id:   orderItemPack.Table_id,
			Created_at: now,
			Updated_at: now,
		}

		if order.Table_id == nil || *order.Table_id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "table_id is required",
			})
			return
		}

		// Create the parent order and obtain its ID.
		orderID := OrderItemOrderCreator(order)

		orderItemsToInsert := make([]interface{}, 0, len(orderItemPack.Order_items))

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		for _, item := range orderItemPack.Order_items {
			item.Order_id = orderID

			if err := validate.Struct(item); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid order item",
					"details": err.Error(),
				})
				return
			}

			item.ID = primitive.NewObjectID()
			item.Order_item_id = item.ID.Hex()

			unitPrice := math.Round(*item.Unit_price*100) / 100
			item.Unit_price = &unitPrice

			item.Created_at = now
			item.Updated_at = now

			orderItemsToInsert = append(
				orderItemsToInsert,
				item,
			)
		}

		result, err := orderItemCollection.InsertMany(
			ctx,
			orderItemsToInsert,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create order items",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Order items created successfully",
			"order_id": orderID,
			"data":     result,
		})
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderItemID := c.Param("orderItem_id")

		if orderItemID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "orderItem_id is required",
			})
			return
		}

		var request struct {
			Quantity   *string  `json:"quantity"`
			Unit_price *float64 `json:"unit_price"`
			Food_id    *string  `json:"food_id"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid order item request body",
			})
			return
		}

		// Build the fields that need to be updated.
		updateFields := bson.M{}

		if request.Quantity != nil {
			switch *request.Quantity {
			case "S", "M", "L":
				updateFields["quantity"] = *request.Quantity
			default:
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "quantity must be S, M, or L",
				})
				return
			}
		}

		if request.Unit_price != nil {
			if *request.Unit_price < 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "unit_price cannot be negative",
				})
				return
			}

			unitPrice := math.Round(
				*request.Unit_price*100,
			) / 100

			updateFields["unit_price"] = unitPrice
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			10*time.Second,
		)
		defer cancel()

		if request.Food_id != nil {
			if *request.Food_id == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "food_id cannot be empty",
				})
				return
			}

			err := foodCollection.FindOne(
				ctx,
				bson.M{
					"food_id": *&request.Food_id,
				},
			)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid Food ID",
				})
				return
			}

			updateFields["food_id"] = *request.Food_id
		}

		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one field must be provided",
			})
			return
		}

		updateFields["updated_at"] = time.Now().UTC()
		update := bson.M{
			"$set": updateFields,
		}

		result, err := orderItemCollection.UpdateOne(
			ctx,
			bson.M{
				"order_item_id": orderItemID,
			},
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update order item",
			})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order item not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Order item updated successfully",
		})
	}
}

func DeleteOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {

		orderItemID := c.Param("orderItem_id")

		if orderItemID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "OrdeItem id is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		result, err := orderItemCollection.DeleteOne(
			ctx,
			bson.M{
				"order_item_id": orderItemID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete Order Item",
			})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order Item Not Found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Order Item deleted successfully",
			"orderItemID": orderItemID,
		})
	}
}

// helpers

func ItemsByOrder(orderID string, ctx context.Context) (OrderItems []primitive.M, err error) {

	matchStage := bson.D{
		{"$match", bson.D{{"order_id", orderID}}},
	}

	lookupFoodStage := bson.D{
		{"$lookup", bson.D{
			{"from", "food"},
			{"localField", "food_id"},
			{"foreignField", "food_id"},
			{"as", "food"},
		}},
	}

	unwindFoodstage := bson.D{
		{"$unwind", bson.D{
			{"path", "$food"},
			{"preserveNullAndEmptyArrays", true},
		}},
	}

	lookupOrderStage := bson.D{
		{"$lookup", bson.D{
			{"from", "$order"},
			{"localField", "order_id"},
			{"foreignField", "order_id"},
			{"as", "order"},
		}},
	}

	unwindOrderStage := bson.D{
		{"$unwind", bson.D{
			{"path", "order"},
			{"preserveNullAndEmptyArrays", true},
		}},
	}

	lookupTableStage := bson.D{
		{"$lookup", bson.D{
			{"from", "table"},
			{"localField", "order.table.id"},
			{"foreignField", "table_id"},
			{"as", "table"},
		}},
	}

	unwindTableStage := bson.D{
		{"$unwind", bson.D{
			{"path", "$table"},
			{"preserveNullAndEmptyArrays", true},
		}},
	}

	projectStage1 := bson.D{
		{"$project", bson.D{
			{"id", 0},
			{"amount", "$food.prive"},
			{"food_name", "$food.name"},
			{"food_image", "$food.image"},
			{"table_number", "$table.table_number"},
			{"table_id", "$table.table_id"},
			{"order_id", "$order.order_id"},
			{"quantity", 1},
		}},
	}

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", bson.D{
				{"order_id", "$order.order_id"},
				{"table_id", "$table_id"},
				{"table_number", "$table_number"},
			}},

			{"payment_due", bson.D{
				{"$sum", "$amount"},
			}},

			{"total_count", bson.D{
				{"$sum", 1},
			}},

			{"order_items", bson.D{
				{"$push", "$$ROOT"},
			}},
		}},
	}

	projectStage2 := bson.D{
		{"$project", bson.D{
			{"id", 0},
			{"order_id", "$_id.order_id"},
			{"table_id", "$_id.table_id"},
			{"table_number", "$_id.table_number"},
			{"payment_due", 1},
			{"total_count", 1},
			{"order_items", 1},
		}},
	}

	mongoPipeline := mongo.Pipeline{
		matchStage,
		lookupFoodStage,
		unwindFoodstage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage1,
		groupStage,
		projectStage2,
	}

	result, err := orderItemCollection.Aggregate(
		ctx,
		mongoPipeline,
	)

	if err != nil {
		return nil, err
	}
	defer result.Close(ctx)

	OrderItems = make([]primitive.M, 0)

	if err := result.All(ctx, &OrderItems); err != nil {
		return nil, err
	}

	return OrderItems, nil
}
