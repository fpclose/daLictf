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
		// 公开模块
		// =======================
		// 用户公开接口
		usersPublic := apiV1.Group("/users")
		{
			usersPublic.POST("/register", controllers.Register)
			usersPublic.POST("/login", controllers.Login)
		}
		// 比赛大屏
		scoreboardRoutes := apiV1.Group("/scoreboard")
		{
			scoreboardRoutes.GET("", controllers.GetScoreboard)
			scoreboardRoutes.GET("/feed", controllers.GetSolveFeed)
		}
		// 比赛基础信息
		contestRoutes := apiV1.Group("/contest")
		{
			contestRoutes.GET("/current", controllers.GetCurrentContest)
			contestRoutes.GET("/status", controllers.GetContestStatus)
		}
		// 学校列表
		apiV1.GET("/schools", controllers.GetSchoolList)
		apiV1.GET("/schools/:id", middlewares.JWTTryAuthMiddleware(), controllers.GetSchoolDetail)
		// 题目类型列表
		apiV1.GET("/question-types", controllers.GetQuestionTypeList)
		apiV1.GET("/question-types/:id", controllers.GetQuestionTypeDetail)

		// =======================
		// 需登录模块
		// =======================
		authRequired := apiV1.Group("/")
		authRequired.Use(middlewares.JWTAuthMiddleware())
		{
			// 用户
			usersAuth := authRequired.Group("/users")
			{
				usersAuth.GET("/:id", controllers.GetUserDetail)
				usersAuth.PUT("/:id", controllers.UpdateUser)
			}

			// 队伍
			teamRoutes := authRequired.Group("/teams")
			{
				teamRoutes.POST("", controllers.CreateTeam)
				teamRoutes.POST("/join", controllers.JoinTeam)
				teamRoutes.POST("/leave", controllers.LeaveTeam)
				teamRoutes.DELETE("/:id", controllers.DisbandTeam)
				teamRoutes.DELETE("/:id/members/:user_id", controllers.KickMember)
				teamRoutes.PUT("/:id", controllers.UpdateTeam)
				teamRoutes.GET("/:id/solves", controllers.GetTeamSolves)
			}

			// 题目
			challengeRoutes := authRequired.Group("/challenges")
			{
				challengeRoutes.GET("", controllers.ListChallenges)
				challengeRoutes.GET("/:id", controllers.GetChallengeDetail)
				challengeRoutes.POST("/:id/submit", controllers.SubmitFlag)
				challengeRoutes.GET("/:id/attachments", controllers.ListAttachments)
			}

			// 附件下载
			attachmentRoutes := authRequired.Group("/attachments")
			{
				attachmentRoutes.GET("/:attachment_id/download", controllers.DownloadAttachment)
			}

			// 动态容器
			containerRoutes := authRequired.Group("/containers")
			{
				containerRoutes.POST("", controllers.CreateContainer)
				containerRoutes.GET("", controllers.ListContainers)
				containerRoutes.PUT("/:id/renew", controllers.RenewContainer)
				containerRoutes.DELETE("/:id", controllers.DestroyContainer)
			}
		}

		// =======================
		// 管理员模块
		// =======================
		adminAPIs := apiV1.Group("/admin")
		adminAPIs.Use(middlewares.JWTAuthMiddleware(), middlewares.RoleAuthMiddleware(models.RoleAdmin))
		{
			// 用户管理
			adminAPIs.GET("/users", controllers.GetUserList)
			adminAPIs.DELETE("/users/:id", controllers.DeleteUser)
			adminAPIs.PUT("/users/:id/status", controllers.UpdateUserStatus)
			adminAPIs.PUT("/users/:id/role", middlewares.RoleAuthMiddleware(models.RoleRootAdmin), controllers.UpdateUserRole)

			// 学校管理
			adminAPIs.POST("/schools", controllers.CreateSchool)
			adminAPIs.PUT("/schools/:id", controllers.UpdateSchool)
			adminAPIs.DELETE("/schools/:id", controllers.DeleteSchool)
			adminAPIs.PUT("/schools/:id/status", controllers.UpdateSchoolStatus)
			adminAPIs.POST("/schools/:id/reset-invitation-code", controllers.ResetInvitationCode)

			// 队伍管理
			adminAPIs.GET("/teams", controllers.AdminGetTeams)
			adminAPIs.PUT("/teams/:id/status", controllers.AdminUpdateTeamStatus)
			adminAPIs.DELETE("/teams/:id", controllers.AdminDeleteTeam)

			// 题目类型管理
			adminAPIs.POST("/question-types", controllers.CreateQuestionType)
			adminAPIs.PUT("/question-types/:id", controllers.UpdateQuestionType)
			adminAPIs.DELETE("/question-types/:id", controllers.DeleteQuestionType)

			// 题目管理
			adminAPIs.POST("/challenges", controllers.CreateChallenge)
			adminAPIs.PUT("/challenges/:id", controllers.UpdateChallenge)
			adminAPIs.DELETE("/challenges/:id", controllers.DeleteChallenge)
			adminAPIs.GET("/challenges", controllers.AdminListChallenges)
			adminAPIs.GET("/challenges/:id", controllers.AdminGetChallengeDetail)

			// 附件管理
			adminAPIs.POST("/challenges/:id/attachments", controllers.AddAttachment)
			adminAPIs.PUT("/attachments/:attachment_id", controllers.UpdateAttachmentStatus)
			adminAPIs.DELETE("/attachments/:attachment_id", controllers.DeleteAttachment)
			adminAPIs.POST("/attachments/:attachment_id/rescan", controllers.RescanAttachment)

			// 动态容器管理
			adminAPIs.GET("/containers/:id/pcap", controllers.GetPcapLog)
			adminAPIs.DELETE("/containers/:id", controllers.AdminDestroyContainer)

			// Flag 审计
			adminAPIs.GET("/flags/logs", controllers.GetFlagLogs)
			adminAPIs.PUT("/flags/:id/suspect", controllers.MarkSuspectSubmission)
			adminAPIs.GET("/flags/compare", controllers.CompareFlagSubmissions)

			// 比赛信息管理
			adminAPIs.POST("/contest", controllers.UpsertContest)
			adminAPIs.POST("/contest/schools", controllers.AddContestSchool)
			adminAPIs.DELETE("/contest/schools/:id", controllers.DeleteContestSchool)
			adminAPIs.POST("/contest/sponsors", controllers.AddContestSponsor)
			adminAPIs.DELETE("/contest/sponsors/:id", controllers.DeleteContestSponsor)
		}
	}

	return r
}
