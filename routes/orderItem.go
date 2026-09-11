package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func OrderItem(router *gin.Engine) {
	router.GET("/orderItems", controller.GetOrderItems())
	router.GET("/orderItems/:orderItem_id", controller.GetOrderItem())
	router.GET("/orderItems-order/:order_id", controller.GetOrderItemsByOrder())
	router.POST("/orderItemss", controller.CreateOrderItem())
	router.PATCH("/orderItems/:orderItem_id", controller.UpdateOrderItem())
	router.DELETE("/orderItems/:orderItem_id", controller.DeleteOrderItem())
}
