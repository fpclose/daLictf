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
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var DockerClient *client.Client

// InitDocker 初始化 Docker 客户端
func InitDocker() {
	var err error
	DockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to connect to Docker daemon: %v", err)
	}
	log.Println("Docker client initialized successfully.")
}

// existsLocally 仅判断本地是否已有该镜像
func existsLocally(ctx context.Context, ref string) (bool, error) {
	_, _, err := DockerClient.ImageInspectWithRaw(ctx, ref)
	if err == nil {
		return true, nil
	}
	if errdefs.IsNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect image %q: %w", ref, err)
}

// encodeRegistryAuth 生成私有仓库认证串；如果不用私有仓库，可以不调用本函数
func encodeRegistryAuth(user, pass, server string) (string, error) {
	ac := registrytypes.AuthConfig{
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

// ensureImage 本地不存在时才去拉取，避免“远程没有但本地有却仍去拉远程”的问题
func ensureImage(ctx context.Context, ref string, registryAuth string) error {
	ok, err := existsLocally(ctx, ref)
	if err != nil {
		return err
	}
	if ok {
		return nil // 本地已存在，跳过拉取
	}

	// 设置超时，避免网络问题卡死
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	rc, err := DockerClient.ImagePull(pullCtx, ref, imagetypes.PullOptions{
		RegistryAuth: registryAuth, // 公有镜像留空；私有镜像传 encodeRegistryAuth 的结果
	})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
	return nil
}

// CreateContainer 创建容器（会自动确保镜像可用）
func CreateContainer(challenge models.Challenge, team models.Team, flag string) (container.CreateResponse, error) {
	ctx := context.Background()
	containerName := fmt.Sprintf("ctf_challenge_%d_%d", team.ID, challenge.ID)

	// 1) 先确保镜像可用：本地有就用本地；没有再拉
	var registryAuth string
	// 私有仓库时放开下面一行并填写凭据
	// registryAuth, _ = encodeRegistryAuth("USERNAME", "PASSWORD", "https://your-registry.example.com")
	if err := ensureImage(ctx, challenge.DockerImage, registryAuth); err != nil {
		return container.CreateResponse{}, fmt.Errorf("Docker API Error: %v", err)
	}

	// 2) 解析端口配置（形如 {"80":"tcp","3306":"tcp"}）
	var ports map[string]string
	exposedPorts := nat.PortSet{}
	if err := json.Unmarshal([]byte(challenge.DockerPorts), &ports); err == nil {
		for p, proto := range ports {
			port, err := nat.NewPort(proto, p)
			if err != nil {
				log.Printf("Warning: Invalid port format '%s:%s' for challenge %d", proto, p, challenge.ID)
				continue
			}
			exposedPorts[port] = struct{}{}
		}
	}

	// 3) 容器与主机配置
	config := &container.Config{
		Image:        challenge.DockerImage,
		Env:          []string{"DALICTF_FLAG=" + flag}, // 若题目镜像读取的是 FLAG，这里改回 "FLAG="+flag
		ExposedPorts: exposedPorts,
	}
	hostConfig := &container.HostConfig{
		AutoRemove:      true, // 容器退出后自动删除
		PublishAllPorts: true, // 自动随机映射所有 ExposedPorts
	}

	// 4) 创建容器
	return DockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
}

// StartContainer 启动容器
func StartContainer(containerID string) error {
	return DockerClient.ContainerStart(context.Background(), containerID, container.StartOptions{})
}

// DestroyContainer 强制删除容器
func DestroyContainer(containerID string) error {
	return DockerClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
}

// GetContainerInfo 获取容器信息（含端口映射）
func GetContainerInfo(containerID string) (types.ContainerJSON, error) {
	return DockerClient.ContainerInspect(context.Background(), containerID)
}

// IsContainerRunning 判断容器是否在运行
func IsContainerRunning(dockerID string) bool {
	info, err := DockerClient.ContainerInspect(context.Background(), dockerID)
	if err != nil {
		return false
	}
	return info.State != nil && info.State.Running
}
