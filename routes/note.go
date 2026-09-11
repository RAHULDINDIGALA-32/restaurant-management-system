package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Note(router *gin.Engine) {
	router.GET("/notes", controller.GetNotes())
	router.GET("/notes/:note_id", controller.GetNote())
	router.POST("/notes", controller.CreateNote())
	router.PATCH("/notes/:note_id", controller.UpdateNote())
	router.DELETE("/notes/:note_id", controller.DeleteNote())
}
