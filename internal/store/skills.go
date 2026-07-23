package store

import (
	"encoding/json"
	"time"
)

type SkillRow struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Instructions string    `json:"instructions"`
	Agent        string    `json:"agent"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (db *DB) CreateSkill(userID, name, description, instructions, agent string, tags []string) (*SkillRow, error) {
	s := &SkillRow{
		ID:           genID(),
		UserID:       userID,
		Name:         name,
		Description:  description,
		Instructions: instructions,
		Agent:        agent,
		Tags:         tags,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	tagsJSON, _ := json.Marshal(tags)
	_, err := db.DB.Exec(
		"INSERT INTO custom_skills (id, user_id, name, description, instructions, agent, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.ID, s.UserID, s.Name, s.Description, s.Instructions, s.Agent, string(tagsJSON), s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) ListSkillsByUser(userID string) ([]*SkillRow, error) {
	rows, err := db.DB.Query(
		"SELECT id, user_id, name, description, instructions, agent, tags, created_at, updated_at FROM custom_skills WHERE user_id = ? ORDER BY name",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*SkillRow
	for rows.Next() {
		var s SkillRow
		var tagsJSON string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &s.Instructions, &s.Agent, &tagsJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &s.Tags)
		skills = append(skills, &s)
	}
	return skills, rows.Err()
}

func (db *DB) DeleteSkill(userID, name string) error {
	_, err := db.DB.Exec("DELETE FROM custom_skills WHERE user_id = ? AND name = ?", userID, name)
	return err
}
