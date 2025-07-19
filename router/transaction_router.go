package routes

import (
	controllers "go-mvc-demo/controller"
	"go-mvc-demo/middleware"

	"github.com/gin-gonic/gin"
)

func TransactionRoutes(r *gin.Engine) {
	auth := r.Group("/", middleware.AuthMiddleware())
	auth.POST("/buy/:id", controllers.BuyGame)
	auth.POST("/rent/:id", controllers.RentGame)
	auth.GET("/rental/check/:id", controllers.CheckActiveRental)
	auth.POST("/recharge", controllers.RechargeCoin)
	auth.GET("/recharge-history", controllers.GetRechargeHistory)
}
