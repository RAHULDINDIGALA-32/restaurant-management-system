package main

import (
	"os"
	"restaurant-management-system/database"
	"restaurant-management-system/middleware"
	"restaurant-management-system/routes"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var foodCollection *mongo.Collection = database.OpenCoection(database.Cient, "food")

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	router := gin.New()
	router.Use(gin.Logger())

	routes.User(router)
	router.Use(middleware.Authentication())

	routes.Menu(router)
	routes.Food(router)
	routes.Table(router)
	routes.Order(router)
	routes.OrderItem(router)
	routes.Note(router)
	routes.Invoice(router)

	router.Run(":" + port)

}
