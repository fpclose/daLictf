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
		// =======================
		// 用户模块
		// =======================
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

		// 管理员 - 用户管理
		adminUsers := apiV1.Group("/admin")
		adminUsers.Use(
			middlewares.JWTAuthMiddleware(),
			middlewares.RoleAuthMiddleware(models.RoleAdmin),
		)
		{
			adminUsers.GET("/users", controllers.GetUserList)
			adminUsers.DELETE("/users/:id", controllers.DeleteUser)
			adminUsers.PUT("/users/:id/status", controllers.UpdateUserStatus)

			// 仅 root_admin 可改角色
			adminUsers.PUT(
				"/users/:id/role",
				middlewares.RoleAuthMiddleware(models.RoleRootAdmin),
				controllers.UpdateUserRole,
			)
		}

		// =======================
		// 学校模块
		// =======================
		schoolRoutes := apiV1.Group("/schools")
		{
			schoolRoutes.GET("", controllers.GetSchoolList)

			// 允许“尝试认证”查看（已登录可带更多信息）
			schoolRoutes.GET("/:id",
				middlewares.JWTTryAuthMiddleware(),
				controllers.GetSchoolDetail,
			)

			// 管理员
			schoolRoutes.POST("",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.CreateSchool,
			)
			schoolRoutes.PUT("/:id",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.UpdateSchool,
			)
			schoolRoutes.DELETE("/:id",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.DeleteSchool,
			)
			schoolRoutes.PUT("/:id/status",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.UpdateSchoolStatus,
			)
			schoolRoutes.POST("/:id/reset-invitation-code",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.ResetInvitationCode,
			)
		}

		// =======================
		// 队伍模块
		// =======================
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

		// 管理员 - 队伍管理
		adminTeamRoutes := apiV1.Group("/admin/teams")
		adminTeamRoutes.Use(
			middlewares.JWTAuthMiddleware(),
			middlewares.RoleAuthMiddleware(models.RoleAdmin),
		)
		{
			adminTeamRoutes.GET("", controllers.AdminGetTeams)
			adminTeamRoutes.PUT("/:id/status", controllers.AdminUpdateTeamStatus)
			adminTeamRoutes.DELETE("/:id", controllers.AdminDeleteTeam)
		}

		// =======================
		// 题目类型模块
		// =======================
		qtRoutes := apiV1.Group("/question-types")
		{
			// 公开
			qtRoutes.GET("", controllers.GetQuestionTypeList)
			qtRoutes.GET("/:id", controllers.GetQuestionTypeDetail)

			// 管理员
			qtRoutes.POST("",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.CreateQuestionType,
			)
			qtRoutes.PUT("/:id",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.UpdateQuestionType,
			)
			qtRoutes.DELETE("/:id",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.DeleteQuestionType,
			)
		}

		// =======================
		// 题目模块
		// =======================
		challengeRoutes := apiV1.Group("/challenges")
		{
			// 用户侧（需要登录；仅返回 state=visible）
			challengeRoutes.GET("",
				middlewares.JWTAuthMiddleware(),
				controllers.ListChallenges,
			)
			challengeRoutes.GET("/:id",
				middlewares.JWTAuthMiddleware(),
				controllers.GetChallengeDetail,
			)
			challengeRoutes.POST("/:id/submit",
				middlewares.JWTAuthMiddleware(),
				controllers.SubmitFlag,
			)

			// 管理员创建题目
			challengeRoutes.POST("",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.CreateChallenge,
			)

			// 附件（外链 JSON & multipart 上传）
			challengeRoutes.POST("/:id/attachments",
				middlewares.JWTAuthMiddleware(),
				middlewares.RoleAuthMiddleware(models.RoleAdmin),
				controllers.AddAttachment,
			)

			// TODO（如有需要）：后续补充管理员更新/删除题目
			// challengeRoutes.PUT("/:id",    middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.UpdateChallenge)
			// challengeRoutes.DELETE("/:id", middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin), controllers.DeleteChallenge)
		}

		// 管理员 - 题目管理（不受可见性限制）
		adminChal := apiV1.Group("/admin/challenges")
		adminChal.Use(
			middlewares.JWTAuthMiddleware(),
			middlewares.RoleAuthMiddleware(models.RoleAdmin),
		)
		{
			// 新增：管理员题目列表 & 详情
			adminChal.GET("", controllers.AdminListChallenges)
			adminChal.GET("/:id", controllers.AdminGetChallengeDetail)

			// 可选：管理员置显/置隐
			// 仅当你实现了 AdminUpdateChallengeState 时再放开
			//adminChal.PUT("/:id/state", controllers.AdminUpdateChallengeState)
		}

		// =======================
		// 附件下载统一网关（需登录）
		// =======================
		attachmentRoutes := apiV1.Group("/attachments")
		{
			attachmentRoutes.GET("/:attachment_id/download",
				middlewares.JWTAuthMiddleware(),
				controllers.DownloadAttachment,
			)
		}
	}

	return r
}
