// file: mappers/challenge_mapper.go
package mappers

import (
	"ISCTF/dto"
	"ISCTF/models"
)

func MapCreateReqToModel(req dto.CreateChallengeReq) models.Challenge {
	return models.Challenge{
		ChallengeName:   req.ChallengeName,
		ChallengeTypeID: req.ChallengeTypeID,
		Author:          req.Author,
		Description:     req.Description,
		Hint:            req.Hint,
		Mode:            models.ChallengeMode(req.Mode),
		StaticFlag:      req.StaticFlag,
		DockerImage:     req.DockerImage,
		DockerPorts:     req.DockerPorts,
		Difficulty:      models.ChallengeDifficulty(req.Difficulty),
		InitialScore:    req.InitialScore,
		MinScore:        req.MinScore,
		CurrentScore:    req.InitialScore, // 初始化为初始分
		DecayRatio:      req.DecayRatio,
	}
}

func MapModelToItemResp(ch models.Challenge) dto.ChallengeItemResp {
	return dto.ChallengeItemResp{
		ID:            ch.ID,
		ChallengeName: ch.ChallengeName,
		Type:          ch.QuestionType.Alias,
		Difficulty:    string(ch.Difficulty),
		Mode:          string(ch.Mode),
		CurrentScore:  ch.CurrentScore,
		SolvedCount:   ch.SolvedCount,
	}
}

func MapModelToDetailResp(ch models.Challenge, atts []models.Attachment) dto.ChallengeDetailResp {
	mini := make([]dto.AttachmentMini, 0, len(atts))
	for _, a := range atts {
		mini = append(mini, dto.AttachmentMini{
			ID:       a.ID,
			FileName: a.FileName,
			Size:     uint64(a.FileSize),
			SHA256:   a.SHA256,
			Status:   string(a.Status),
		})
	}
	return dto.ChallengeDetailResp{
		ID:            ch.ID,
		ChallengeName: ch.ChallengeName,
		Author:        ch.Author,
		Description:   ch.Description,
		Hint:          ch.Hint,
		Mode:          string(ch.Mode),
		Difficulty:    string(ch.Difficulty),
		Attachments:   mini,
		CurrentScore:  ch.CurrentScore,
		SolvedCount:   ch.SolvedCount,
	}
}
