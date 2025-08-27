// file: dto/challenge.go
package dto

import "strings"

// ========== 请求 DTO ==========

type CreateChallengeReq struct {
	// 规范字段（snake_case）
	ChallengeName   string  `json:"challenge_name"`
	ChallengeTypeID uint32  `json:"challenge_type_id"`
	Author          string  `json:"author"`
	Description     string  `json:"description"`
	Hint            string  `json:"hint"`
	Mode            string  `json:"mode"` // static / dynamic
	StaticFlag      string  `json:"static_flag"`
	DockerImage     string  `json:"docker_image"`
	DockerPorts     string  `json:"docker_ports"`
	Difficulty      string  `json:"difficulty"` // easy / medium / hard
	InitialScore    uint    `json:"initial_score"`
	MinScore        uint    `json:"min_score"`
	DecayRatio      float32 `json:"decay_ratio"`

	// 仅用于兼容旧客户端（camelCase / 大小写变体），注意：所有别名都与上面 tag 不重复
	ChallengeNameCamel    string  `json:"challengeName"`
	ChallengeTypeIDCamel  uint32  `json:"challengeTypeId"`
	ChallengeTypeIDCamel2 uint32  `json:"challengeTypeID"`
	StaticFlagCamel       string  `json:"staticFlag"`
	DockerImageCamel      string  `json:"dockerImage"`
	DockerPortsCamel      string  `json:"dockerPorts"`
	DifficultyCamel       string  `json:"difficulty"`
	InitialScoreCamel     uint    `json:"initialScore"`
	MinScoreCamel         uint    `json:"minScore"`
	DecayRatioCamel       float32 `json:"decayRatio"`
}

// Normalize: 将 camelCase 别名归一化到 snake_case，并做轻量默认值处理
func (r *CreateChallengeReq) Normalize() {
	// 别名归一化
	if r.ChallengeName == "" && r.ChallengeNameCamel != "" {
		r.ChallengeName = r.ChallengeNameCamel
	}
	if r.ChallengeTypeID == 0 {
		if r.ChallengeTypeIDCamel != 0 {
			r.ChallengeTypeID = r.ChallengeTypeIDCamel
		} else if r.ChallengeTypeIDCamel2 != 0 {
			r.ChallengeTypeID = r.ChallengeTypeIDCamel2
		}
	}
	if r.StaticFlag == "" && r.StaticFlagCamel != "" {
		r.StaticFlag = r.StaticFlagCamel
	}
	if r.DockerImage == "" && r.DockerImageCamel != "" {
		r.DockerImage = r.DockerImageCamel
	}
	if r.DockerPorts == "" && r.DockerPortsCamel != "" {
		r.DockerPorts = r.DockerPortsCamel
	}
	if r.Difficulty == "" && r.DifficultyCamel != "" {
		r.Difficulty = r.DifficultyCamel
	}
	if r.InitialScore == 0 && r.InitialScoreCamel != 0 {
		r.InitialScore = r.InitialScoreCamel
	}
	if r.MinScore == 0 && r.MinScoreCamel != 0 {
		r.MinScore = r.MinScoreCamel
	}
	if r.DecayRatio == 0 && r.DecayRatioCamel != 0 {
		r.DecayRatio = r.DecayRatioCamel
	}

	// 清洗/默认值
	r.ChallengeName = strings.TrimSpace(r.ChallengeName)
	r.Author = strings.TrimSpace(r.Author)
	r.Description = strings.TrimSpace(r.Description)
	r.Mode = strings.ToLower(strings.TrimSpace(r.Mode))
	r.Difficulty = strings.ToLower(strings.TrimSpace(r.Difficulty))

	if r.Difficulty == "" {
		r.Difficulty = "medium"
	}
	if r.DecayRatio == 0 {
		r.DecayRatio = 0.1
	}
}

type UpdateChallengeReq struct {
	State       *string `json:"state"` // visible/hidden
	Hint        *string `json:"hint"`
	Difficulty  *string `json:"difficulty"`
	Mode        *string `json:"mode"`
	StaticFlag  *string `json:"static_flag"`
	DockerImage *string `json:"docker_image"`
	DockerPorts *string `json:"docker_ports"`
}

type SubmitFlagReq struct {
	Flag      string `json:"flag"`
	FlagCamel string `json:"Flag"`
}

func (r *SubmitFlagReq) Normalize() {
	if r.Flag == "" && r.FlagCamel != "" {
		r.Flag = r.FlagCamel
	}
}

// ========== 响应 DTO ==========

type ChallengeItemResp struct {
	ID            uint32 `json:"id"`
	ChallengeName string `json:"challenge_name"`
	Type          string `json:"type"`
	Difficulty    string `json:"difficulty"`
	Mode          string `json:"mode"`
	CurrentScore  uint   `json:"current_score"`
	SolvedCount   uint   `json:"solved_count"`
}

type AttachmentMini struct {
	ID       uint64 `json:"id"`
	FileName string `json:"file_name"`
	Size     uint64 `json:"size"`
	SHA256   string `json:"sha256"`
	Status   string `json:"status"`
}

type ChallengeDetailResp struct {
	ID            uint32           `json:"id"`
	ChallengeName string           `json:"challenge_name"`
	Author        string           `json:"author"`
	Description   string           `json:"description"`
	Hint          string           `json:"hint"`
	Mode          string           `json:"mode"`
	Difficulty    string           `json:"difficulty"`
	Attachments   []AttachmentMini `json:"attachments"`
	CurrentScore  uint             `json:"current_score"`
	SolvedCount   uint             `json:"solved_count"`
}

// ====== Admin 专用响应 DTO ======

type AdminChallengeItemResp struct {
	ID            uint32 `json:"id"`
	ChallengeName string `json:"challenge_name"`
	Type          string `json:"type"`
	Difficulty    string `json:"difficulty"`
	Mode          string `json:"mode"`
	State         string `json:"state"`
	CurrentScore  uint   `json:"current_score"`
	SolvedCount   uint   `json:"solved_count"`
	UpdatedAt     string `json:"updated_at"`
}

type AdminAttachmentMini struct {
	ID       uint64 `json:"id"`
	FileName string `json:"file_name"`
	Size     uint64 `json:"size"`
	SHA256   string `json:"sha256"`
	Status   string `json:"status"`
	Storage  string `json:"storage"`
}

type AdminChallengeDetailResp struct {
	ID            uint32                `json:"id"`
	ChallengeName string                `json:"challenge_name"`
	Type          string                `json:"type"`
	Author        string                `json:"author"`
	Description   string                `json:"description"`
	Hint          string                `json:"hint"`
	Mode          string                `json:"mode"`
	Difficulty    string                `json:"difficulty"`
	State         string                `json:"state"`
	StaticFlag    string                `json:"static_flag,omitempty"` // 视需要也可不返回
	DockerImage   string                `json:"docker_image,omitempty"`
	DockerPorts   string                `json:"docker_ports,omitempty"`
	CurrentScore  uint                  `json:"current_score"`
	InitialScore  uint                  `json:"initial_score"`
	MinScore      uint                  `json:"min_score"`
	DecayRatio    float32               `json:"decay_ratio"`
	SolvedCount   uint                  `json:"solved_count"`
	Attachments   []AdminAttachmentMini `json:"attachments"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
}
