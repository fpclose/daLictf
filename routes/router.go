// file: routes/router.go
package routes

import (
	"ISCTF/controllers"
	"ISCTF/middlewares"
	"ISCTF/models"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	apiV1 := r.Group("/api/v1")
	{
		// --- 用户、学校、队伍等路由保持不变 ---
		usersPublic := apiV1.Group("/users")
		{
			usersPublic.POST("/register", controllers.Register)
			usersPublic.POST("/login", controllers.Login)
		}
		usersAuth := apiV1.Group("/users")
		usersAuth.Use(middlewares.JWTAuthMiddleware())
		{
			usersAuth.GET("/:id", controllers.GetUserDetail)
			usersAuth.PUT("/:id", controllers.UpdateUser)
		}
		adminRoutes := apiV1.Group("/admin")
		adminRoutes.Use(middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin))
		{
			adminRoutes.GET("/users", controllers.GetUserList)
			adminRoutes.DELETE("/users/:id", controllers.DeleteUser)
			adminRoutes.PUT("/users/:id/status", controllers.UpdateUserStatus)
			adminRoutes.PUT("/users/:id/role", middlewares.RoleAuthMiddleware(models.RoleRootAdmin), controllers.UpdateUserRole)
		}
		schoolRoutes := apiV1.Group("/schools")
		{
			schoolRoutes.GET("", controllers.GetSchoolList)
			schoolRoutes.GET("/:id", middlewares.JWTTryAuthMiddleware(), controllers.GetSchoolDetail)
			schoolRoutes.POST("", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.CreateSchool)
			schoolRoutes.PUT("/:id", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.UpdateSchool)
			schoolRoutes.DELETE("/:id", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.DeleteSchool)
			schoolRoutes.PUT("/:id/status", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.UpdateSchoolStatus)
			schoolRoutes.POST("/:id/reset-invitation-code", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.ResetInvitationCode)
		}
		teamRoutes := apiV1.Group("/teams")
		teamRoutes.Use(middlewares.JWTAuthMiddleware())
		{
			teamRoutes.POST("", controllers.CreateTeam)
			teamRoutes.POST("/join", controllers.JoinTeam)
			teamRoutes.POST("/leave", controllers.LeaveTeam)
			teamRoutes.DELETE("/:id", controllers.DisbandTeam)
			teamRoutes.DELETE("/:id/members/:user_id", controllers.KickMember)
			teamRoutes.PUT("/:id", controllers.UpdateTeam)
		}
		adminTeamRoutes := apiV1.Group("/admin/teams")
		adminTeamRoutes.Use(middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin))
		{
			adminTeamRoutes.GET("", controllers.AdminGetTeams)
			adminTeamRoutes.PUT("/:id/status", controllers.AdminUpdateTeamStatus)
			adminTeamRoutes.DELETE("/:id", controllers.AdminDeleteTeam)
		}

		// =======================================================
		//  ↓↓↓ 新增：题目类型模块路由 ↓↓↓
		// =======================================================
		questionTypeRoutes := apiV1.Group("/question-types")
		{
			// 公开接口
			questionTypeRoutes.GET("", controllers.GetQuestionTypeList)
			questionTypeRoutes.GET("/:id", controllers.GetQuestionTypeDetail)

			// 管理员接口
			questionTypeRoutes.POST("", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.CreateQuestionType)
			questionTypeRoutes.PUT("/:id", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.UpdateQuestionType)
			questionTypeRoutes.DELETE("/:id", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.DeleteQuestionType)
		}

		// --- 题目模块路由 ---
		challengeRoutes := apiV1.Group("/challenges")
		{
			// 用户接口
			challengeRoutes.GET("", middlewares.JWTAuthMiddleware(), controllers.ListChallenges)
			challengeRoutes.GET("/:id", middlewares.JWTAuthMiddleware(), controllers.GetChallengeDetail)
			challengeRoutes.POST("/:id/submit", middlewares.JWTAuthMiddleware(), controllers.SubmitFlag)

			// 管理员接口
			challengeRoutes.POST("", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.CreateChallenge)
			// ... Update, Delete 路由待添加 ...

			// 附件相关
			challengeRoutes.POST("/:id/attachments", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.AddAttachment)
		}

		// --- 附件下载统一网关 ---
		attachmentRoutes := apiV1.Group("/attachments")
		{
			attachmentRoutes.GET("/:attachment_id/download", middlewares.JWTAuthMiddleware(), controllers.DownloadAttachment)
		}
	}

	return r
}
