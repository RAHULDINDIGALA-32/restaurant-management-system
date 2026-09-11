package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func User(router *gin.Engine) {
	
	router.GET("/users", controller.GetUsers())
	router.GET("/users/:user_id", controller.GetUser())
	router.POST("/users/signup", controller.SignUp())
	router.POST("/users/login", controller.Login())
	router.PATCH("/users/:user_id", controller.UpdateUser())
	router.DELETE("/users/:user_id", controller.DeleteUser())
}
