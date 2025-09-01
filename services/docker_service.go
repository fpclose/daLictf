// file: services/docker_service.go
package services

import (
	"ISCTF/models"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

var DockerClient *client.Client

// InitDocker 初始化 Docker 客户端并检查 Swarm 状态
func InitDocker() {
	var err error
	DockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to connect to Docker daemon: %v", err)
	}

	info, err := DockerClient.Info(context.Background())
	if err != nil {
		log.Fatalf("Failed to get Docker info: %v", err)
	}

	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		log.Fatalf("Docker is not running in Swarm mode. Please run 'docker swarm init'.")
	}

	log.Println("Docker client initialized and connected to Swarm cluster.")
}

// encodeRegistryAuth 生成私有仓库认证串
func encodeRegistryAuth(user, pass, server string) (string, error) {
	ac := registry.AuthConfig{
		Username:      user,
		Password:      pass,
		ServerAddress: server,
	}
	b, err := json.Marshal(ac)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// ensureImage 确保镜像在 Swarm 集群中可用
func ensureImage(ctx context.Context, ref string, registryAuth string) error {
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	rc, err := DockerClient.ImagePull(pullCtx, ref, imagetypes.PullOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

// CreateService 在 Docker Swarm 中创建一个服务来替代单个容器
func CreateService(challenge models.Challenge, team models.Team, flag string) (string, error) {
	ctx := context.Background()
	// 使用时间戳确保服务名唯一，避免冲突
	serviceName := fmt.Sprintf("ctf-%d-%d-%d", team.ID, challenge.ID, time.Now().UnixNano())

	// 确保镜像可用 (在生产环境中，建议提前在所有 Swarm Node 上拉取镜像)
	// var registryAuth string
	// if err := ensureImage(ctx, challenge.DockerImage, registryAuth); err != nil {
	// 	return "", fmt.Errorf("ensure image failed: %v", err)
	// }

	// 1. 解析端口配置，形如 "80,3306"
	var portConfigs []swarm.PortConfig
	ports := strings.Split(challenge.DockerPorts, ",")
	for _, p := range ports {
		port, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
		if err != nil {
			log.Printf("Warning: Invalid port format '%s' for challenge %d", p, challenge.ID)
			continue
		}
		portConfigs = append(portConfigs, swarm.PortConfig{
			Protocol:    swarm.PortConfigProtocolTCP,
			TargetPort:  uint32(port),
			PublishMode: swarm.PortConfigPublishModeIngress, // 使用随机端口模式
		})
	}

	// 2. 定义服务规格 (ServiceSpec)
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: serviceName,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: challenge.DockerImage,
				Env:   []string{"DALICTF_FLAG=" + flag},
			},
			Resources: &swarm.ResourceRequirements{
				Limits: &swarm.Limit{
					MemoryBytes: 256 * 1024 * 1024, // 限制内存 256MB
					NanoCPUs:    500000000,         // 限制 CPU 0.5 Core
				},
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Ports: portConfigs,
		},
	}

	// 3. 创建服务
	createOpts := types.ServiceCreateOptions{}
	serviceResp, err := DockerClient.ServiceCreate(ctx, serviceSpec, createOpts)
	if err != nil {
		return "", err
	}

	return serviceResp.ID, nil
}

// DestroyService 销毁一个服务
func DestroyService(serviceID string) error {
	return DockerClient.ServiceRemove(context.Background(), serviceID)
}

// GetServiceInfo 获取服务信息
func GetServiceInfo(serviceID string) (swarm.Service, []byte, error) {
	return DockerClient.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
}

// IsServiceRunning 检查服务是否仍在运行
func IsServiceRunning(serviceID string) bool {
	_, _, err := GetServiceInfo(serviceID)
	return err == nil
}
