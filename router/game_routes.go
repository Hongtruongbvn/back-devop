package routes

import (
	controllers "go-mvc-demo/controller"
	"go-mvc-demo/middleware"

	"github.com/gin-gonic/gin"
)

func GameRoutes(r *gin.Engine) {
	game := r.Group("/games")
	{
		// Public routess
		game.GET("/", controllers.GetGames)
		game.GET("/:id", controllers.GetGameByID)
		game.GET("/fetch", controllers.FetchAndSaveGames)
		game.GET("/fetch-games", controllers.FetchGamesByPage)

		// Protected routes - Admin only
		game.POST("/", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.CreateGame)
		game.DELETE("/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.DeleteGame)
	}

	// Optional: You can group user-related routes elsewhere
	user := r.Group("/user")
	user.Use(middleware.AuthMiddleware())
	{
		user.GET("/my-purchases", controllers.GetPurchasedGames)
		user.GET("/my-rentals", controllers.GetRentedGames)
	}
}
