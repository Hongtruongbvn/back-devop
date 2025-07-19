package routes

import (
	controllers "go-mvc-demo/controller"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	user := r.Group("/users")
	{
		user.GET("/", controllers.GetUsers)
		user.GET("/:id", controllers.GetUserByID)
		user.POST("/", controllers.CreateUser)
		user.PUT("/:id", controllers.UpdateUser)
		user.DELETE("/:id", controllers.DeleteUser)
		user.PUT("/:id/promote", controllers.PromoteUserToAdmin)
		user.POST("/forgot-password", controllers.ForgotPasswordHandler)
		user.POST("/reset-password", controllers.ResetPasswordHandler)
	}
}
