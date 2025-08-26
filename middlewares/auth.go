// file: middlewares/auth.go
package middlewares

import (
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// JWTAuthMiddleware 验证用户是否登录
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			utils.Error(c, 4001, "请求头中 Authorization 为空")
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			utils.Error(c, 4002, "Authorization 格式有误")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Error(c, 4003, "无效的 Token")
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// RoleAuthMiddleware 验证用户角色权限
func RoleAuthMiddleware(requiredRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, exists := c.Get("user_role")
		if !exists {
			utils.Error(c, 5001, "无法获取用户角色信息")
			c.Abort()
			return
		}

		role := roleAny.(models.UserRole)

		hasPermission := false
		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				hasPermission = true
				break
			}
		}

		// root_admin 拥有所有权限
		if role == models.RoleRootAdmin {
			hasPermission = true
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"code": 4003, "msg": "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// JWTTryAuthMiddleware 尝试解析Token，即使失败也继续执行
func JWTTryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.Next() // 没有Token，直接进入下一个处理函数
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.Next() // Token格式错误，也直接进入
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err == nil {
			// Token有效，将用户信息放入上下文
			c.Set("user_id", claims.UserID)
			c.Set("user_role", claims.Role)
		}

		c.Next()
	}
}
