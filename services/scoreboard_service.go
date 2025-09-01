// file: services/scoreboard_service.go
package services

import (
	"ISCTF/database"
	"ISCTF/models"
	"gorm.io/gorm"
	"log"
	"time"
)

// UpdateScoreboardCache 是核心函数，用于重新计算并更新整个排行榜
func UpdateScoreboardCache() {
	log.Println("Starting to update scoreboard cache...")

	// 辅助结构体，用于从原始解题记录中聚合数据
	type TeamScore struct {
		TeamID        uint32
		TotalScore    uint
		LastSolveTime time.Time
		Track         models.UserTrack
		TeamName      string
		SchoolName    *string
	}

	var teamScores []TeamScore
	// 通过 JOIN 查询和 GROUP BY 聚合，一次性计算出所有队伍的总分和最后解题时间
	database.DB.Table("dalictf_problem_solving_record r").
		Select("r.team_id, SUM(r.score) as total_score, MAX(r.solving_time) as last_solve_time, t.track, t.team_name, s.school_name").
		Joins("JOIN dalictf_team t ON r.team_id = t.id").
		Joins("LEFT JOIN dalictf_school s ON t.school_id = s.id").
		Group("r.team_id, t.track, t.team_name, s.school_name").
		Order("total_score desc, last_solve_time asc").
		Scan(&teamScores)

	// 在事务中更新缓存表，保证数据一致性
	database.DB.Transaction(func(tx *gorm.DB) error {
		// 先清空旧的排行榜数据
		if err := tx.Exec("DELETE FROM dalictf_scoreboard").Error; err != nil {
			return err
		}

		// 按赛道分别计算排名
		rankCounters := make(map[models.ScoreboardTrack]uint)
		var overallRank uint = 0

		for _, ts := range teamScores {
			overallRank++
			track := models.ScoreboardTrack(ts.Track)
			rankCounters[track]++

			// 写入分赛道排名
			entry := models.Scoreboard{
				TeamID:        ts.TeamID,
				TeamName:      ts.TeamName,
				SchoolName:    ts.SchoolName,
				Track:         track,
				Score:         ts.TotalScore,
				LastSolveTime: &ts.LastSolveTime,
				Rank:          rankCounters[track],
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}

			// 写入总榜排名
			overallEntry := models.Scoreboard{
				TeamID:        ts.TeamID,
				TeamName:      ts.TeamName,
				SchoolName:    ts.SchoolName,
				Track:         models.TrackOverall,
				Score:         ts.TotalScore,
				LastSolveTime: &ts.LastSolveTime,
				Rank:          overallRank,
			}
			if err := tx.Create(&overallEntry).Error; err != nil {
				return err
			}
		}
		return nil
	})

	// 更新数据库后，清空所有与排行榜相关的 Redis 缓存，确保下次查询获取最新数据
	keys, err := database.RDB.Keys(database.Ctx, "scoreboard:*").Result()
	if err == nil && len(keys) > 0 {
		database.RDB.Del(database.Ctx, keys...)
		log.Printf("Cleared %d scoreboard cache keys from Redis.", len(keys))
	}

	log.Println("Scoreboard cache updated successfully.")
}

// AddSolveToFeed 将一条新的解题记录添加到动态缓存中
func AddSolveToFeed(solve models.Submission, challenge models.Challenge, team models.Team) {
	var schoolName *string
	if team.School != nil {
		schoolName = &team.School.SchoolName
	}

	feedEntry := models.SolveFeed{
		ChallengeID:   solve.ChallengeID,
		ChallengeName: challenge.ChallengeName,
		TeamID:        solve.TeamID,
		TeamName:      team.TeamName,
		SchoolName:    schoolName,
		Score:         solve.Score,
		SolvingTime:   solve.SolvingTime,
	}

	database.DB.Create(&feedEntry)

	// (可选) 清理旧的记录，保持表的大小
	var count int64
	database.DB.Model(&models.SolveFeed{}).Count(&count)
	if count > 5000 { // 保留最新的 5000 条
		database.DB.Exec("DELETE FROM dalictf_solve_feed ORDER BY solving_time asc LIMIT ?", count-5000)
	}
}
