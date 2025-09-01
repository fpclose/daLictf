// file: controllers/container_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/services"
	"ISCTF/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

// CreateContainer 申请容器 (已修复重复申请问题)
func CreateContainer(c *gin.Context) {
	var req struct {
		ChallengeID uint32 `json:"challenge_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)

	var userTeam models.TeamMember
	if err := database.DB.Where("user_id = ?", userID).First(&userTeam).Error; err != nil {
		utils.Error(c, 3005, "你尚未加入任何队伍")
		return
	}
	var team models.Team
	database.DB.First(&team, userTeam.TeamID)
	if team.TeamStatus == models.TeamStatusBanned {
		utils.Error(c, 4003, "队伍已被封禁，无法申请容器")
		return
	}

	var challenge models.Challenge
	if err := database.DB.First(&challenge, req.ChallengeID).Error; err != nil {
		utils.Error(c, 4004, "题目不存在")
		return
	}
	if challenge.Mode != models.ChallengeModeDynamic {
		utils.Error(c, 1002, "该题目不是动态容器题目")
		return
	}

	// 修正：检查是否已为该题目申请了正在运行的容器
	var existingContainer models.Container
	err := database.DB.Where("team_id = ? AND challenge_id = ? AND state = ?", team.ID, challenge.ID, models.ContainerStateRunning).First(&existingContainer).Error
	if err == nil {
		utils.Error(c, 7004, "You already have a running container for this challenge")
		return
	}

	var runningCount int64
	database.DB.Model(&models.Container{}).Where("team_id = ? AND state = ?", team.ID, models.ContainerStateRunning).Count(&runningCount)
	if runningCount >= 2 {
		utils.Error(c, 7001, fmt.Sprintf("Team already has %d running containers", runningCount))
		return
	}

	var dynamicFlag string
	for {
		dynamicFlag = utils.GenerateDynamicFlag()
		var count int64
		database.DB.Model(&models.Container{}).Where("container_flag = ?", dynamicFlag).Count(&count)
		if count == 0 {
			break
		}
	}

	// 调用新的 CreateService 函数，它会返回 Service ID
	serviceID, err := services.CreateService(challenge, team, dynamicFlag)
	if err != nil {
		utils.Error(c, 5000, "Docker API Error: "+err.Error())
		return
	}

	now := time.Now()
	newContainer := models.Container{
		DockerID:      serviceID, // 数据库中保存 Service ID
		ChallengeID:   challenge.ID,
		TeamID:        team.ID,
		ContainerName: fmt.Sprintf("ctf-service-%d-%d", team.ID, challenge.ID),
		DockerImage:   challenge.DockerImage,
		DockerPorts:   challenge.DockerPorts,
		ContainerFlag: dynamicFlag,
		State:         models.ContainerStateRunning,
		StartTime:     now,
		EndTime:       now.Add(1 * time.Hour),
	}
	if err := database.DB.Create(&newContainer).Error; err != nil {
		_ = services.DestroyService(serviceID) // 如果数据库保存失败，则销毁服务
		utils.Error(c, 5000, "Failed to save container record: "+err.Error())
		return
	}

	// 获取 Swarm 服务的端口信息
	serviceInfo, _, err := services.GetServiceInfo(serviceID)
	if err != nil {
		log.Printf("Warning: failed to inspect service %s to get port mapping: %v", serviceID, err)
		utils.Error(c, 5000, "Container started but failed to get connection info.")
		return
	}

	connectionInfo := make(map[string]string)
	// =================================================================================
	// [重要] 请将这里的 IP 地址替换为您的 Docker Swarm 集群任一节点的公网或内网 IP
	// =================================================================================
	swarmNodeIP := "127.0.0.1"
	for _, port := range serviceInfo.Endpoint.Ports {
		connectionInfo[strconv.Itoa(int(port.TargetPort))] = fmt.Sprintf("%s:%d", swarmNodeIP, port.PublishedPort)
	}

	utils.Success(c, "Container created successfully", gin.H{
		"container_id":    newContainer.ID,
		"connection_info": connectionInfo,
		"end_time":        newContainer.EndTime.Format("2006-01-02 15:04:05"),
	})
}

// DestroyContainer 销毁容器 (已增加权限校验和状态检查)
func DestroyContainer(c *gin.Context) {
	containerID, _ := strconv.Atoi(c.Param("id"))

	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)
	roleAny, _ := c.Get("user_role")
	userRole := roleAny.(models.UserRole)

	var container models.Container
	if err := database.DB.First(&container, containerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(c, 404, "Container not found")
			return
		}
		utils.Error(c, 5000, "Database error while fetching container")
		return
	}

	// 权限校验：非管理员只能删除自己队伍的容器
	if userRole != models.RoleAdmin && userRole != models.RoleRootAdmin {
		var userTeam models.TeamMember
		if err := database.DB.Where("user_id = ?", userID).First(&userTeam).Error; err != nil || userTeam.TeamID != container.TeamID {
			utils.Error(c, 403, "Permission denied: you can only destroy your own team's containers")
			return
		}
	}

	// 状态检查：如果容器已经是 destroyed 状态，则直接返回成功
	if container.State == models.ContainerStateDestroyed {
		utils.Success(c, "Container already destroyed", nil)
		return
	}

	// 即使容器已停止或销毁，也尝试清理，并更新数据库状态
	if container.State == models.ContainerStateRunning {
		if err := services.DestroyService(container.DockerID); err != nil {
			fmt.Printf("Warning: failed to destroy docker service %s: %v\n", container.DockerID, err)
		}
	}

	container.State = models.ContainerStateDestroyed
	database.DB.Save(&container)

	utils.Success(c, "Container destroyed successfully", nil)
}

// ListContainers 查询队伍容器列表
func ListContainers(c *gin.Context) {
	teamIDStr := c.Query("team_id")
	if teamIDStr == "" {
		utils.Error(c, 1001, "缺少 team_id 参数")
		return
	}
	teamID, _ := strconv.Atoi(teamIDStr)

	var containers []models.Container
	database.DB.Where("team_id = ?", teamID).Find(&containers)

	type ContainerInfo struct {
		ContainerID   uint32 `json:"container_id"`
		ChallengeID   uint32 `json:"challenge_id"`
		ChallengeName string `json:"challenge_name"`
		State         string `json:"state"`
		DockerPorts   string `json:"docker_ports"`
		EndTime       string `json:"end_time"`
	}

	var result []ContainerInfo
	for i := range containers {
		if containers[i].State == models.ContainerStateRunning {
			if !services.IsServiceRunning(containers[i].DockerID) {
				containers[i].State = models.ContainerStateDestroyed
				database.DB.Save(&containers[i])
			}
		}

		var chal models.Challenge
		database.DB.Select("challenge_name").First(&chal, containers[i].ChallengeID)
		result = append(result, ContainerInfo{
			ContainerID:   containers[i].ID,
			ChallengeID:   containers[i].ChallengeID,
			ChallengeName: chal.ChallengeName,
			State:         string(containers[i].State),
			DockerPorts:   containers[i].DockerPorts,
			EndTime:       containers[i].EndTime.Format("2006-01-02 15:04:05"),
		})
	}

	utils.Success(c, "success", result)
}

// RenewContainer 续期容器
func RenewContainer(c *gin.Context) {
	containerID, _ := strconv.Atoi(c.Param("id"))

	var container models.Container
	if err := database.DB.First(&container, containerID).Error; err != nil {
		utils.Error(c, 4004, "容器不存在")
		return
	}

	if container.ExtendedCount >= 3 {
		utils.Error(c, 7002, "Renewal limit reached")
		return
	}
	if container.State != models.ContainerStateRunning {
		utils.Error(c, 7003, "Container is not running")
		return
	}

	var req struct {
		ExtraMinutes uint `json:"extra_minutes"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.ExtraMinutes == 0 {
		req.ExtraMinutes = 30
	}

	container.EndTime = container.EndTime.Add(time.Duration(req.ExtraMinutes) * time.Minute)
	container.ExtendedCount++

	database.DB.Save(&container)

	utils.Success(c, "Container renewed successfully", gin.H{
		"container_id":   container.ID,
		"end_time":       container.EndTime.Format("2006-01-02 15:04:05"),
		"extended_count": container.ExtendedCount,
	})
}

// GetPcapLog 管理员查询抓包日志
func GetPcapLog(c *gin.Context) {
	containerID, _ := strconv.Atoi(c.Param("id"))

	var container models.Container
	if err := database.DB.First(&container, containerID).Error; err != nil {
		utils.Error(c, 4004, "容器不存在")
		return
	}

	utils.Success(c, "success", gin.H{
		"container_id":    container.ID,
		"pcap_path":       container.PcapPath,
		"analysis_result": container.AnalysisResult,
	})
}

// AdminDestroyContainer 管理员强制销毁容器
func AdminDestroyContainer(c *gin.Context) {
	containerID, _ := strconv.Atoi(c.Param("id"))

	var container models.Container
	if err := database.DB.First(&container, containerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Error(c, 404, "Container not found")
			return
		}
		utils.Error(c, 5000, "Database error while fetching container")
		return
	}

	// 状态检查：如果容器已经是 destroyed 状态，则直接返回成功
	if container.State == models.ContainerStateDestroyed {
		utils.Success(c, "Container already destroyed", nil)
		return
	}

	if container.State == models.ContainerStateRunning {
		if err := services.DestroyService(container.DockerID); err != nil {
			// 记录警告，但不阻塞流程，因为容器可能已被手动删除
			fmt.Printf("Warning: failed to destroy docker service %s by admin: %v\n", container.DockerID, err)
		}
	}

	// 更新数据库状态
	container.State = models.ContainerStateDestroyed
	database.DB.Save(&container)

	utils.Success(c, "Container destroyed successfully by admin", nil)
}
