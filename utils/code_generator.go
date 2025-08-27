// file: utils/code_generator.go
package utils

import (
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"strings"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateInvitationCode 生成指定长度的随机邀请码
func GenerateInvitationCode(length int) string {
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

// GenerateDynamicFlag 生成动态 Flag
func GenerateDynamicFlag() string {
	part1 := strings.Replace(uuid.New().String(), "-", "", -1)[:12]
	part2 := strings.Replace(uuid.New().String(), "-", "", -1)[:12]
	part3 := strings.Replace(uuid.New().String(), "-", "", -1)[:12]
	return fmt.Sprintf("ISCTF{%s-%s-%s}", part1, part2, part3)
}
