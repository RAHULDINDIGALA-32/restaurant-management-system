package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Order(router *gin.Engine) {
	router.GET("/orders", controller.GetOrders())
	router.GET("/orders/:order_id", controller.GetOrder())
	router.POST("/orders", controller.CreateOrder())
	router.PATCH("/orders/:order_id", controller.UpdateOrder())
	router.DELETE("/orders/:order_id", controller.DeleteOrder())
}
