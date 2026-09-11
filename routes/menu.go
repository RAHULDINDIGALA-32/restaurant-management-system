package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Menu(router *gin.Engine) {
	router.GET("/menus", controller.GetMenus())
	router.GET("/menus/:menu_id", controller.GetMenu())
	router.POST("/menus", controller.CreateMenu())
	router.PATCH("/menus/:menu_id", controller.UpdateMenu())
	router.DELETE("/menus/:menu_id", controller.DeleteMenu())
}
