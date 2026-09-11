package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Table(router *gin.Engine) {
	router.GET("/tables", controller.GetTables())
	router.GET("/tables/:table_id", controller.GetTable())
	router.POST("/tables", controller.CreateTable())
	router.PATCH("/tables/:table_id", controller.UpdateTable())
	router.DELETE("/tables/:table_id", controller.DeleteTable())
}
