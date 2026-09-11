package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Food(router *gin.Engine) {

	router.GET("/foods", controller.GetFoods())
	router.GET("/foods/:food_id", controller.GetFood())
	router.POST("/foods", controller.CreateFood())
	router.PATCH("/foods/:food_id", controller.UpdateFood())
	router.DELETE("/foods/:food_id", controller.DeleteFood())
}
