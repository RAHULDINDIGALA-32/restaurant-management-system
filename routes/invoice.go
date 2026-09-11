package routes

import (
	controller "restaurant-management-system/controllers"

	"github.com/gin-gonic/gin"
)

func Invoice(router *gin.Engine) {
	router.GET("/invoices", controller.GetInvoices())
	router.GET("/invoices/:invoice_id", controller.GetInvoice())
	router.POST("/invoices", controller.CreateInvoice())
	router.PATCH("/invoices/:invoice_id", controller.UpdateInvoice())
	router.DELETE("/invoices/:invoice_id", controller.DeleteInvoice())
}
