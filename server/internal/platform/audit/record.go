package audit

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID           string    `gorm:"primaryKey"`
	ActorID      string    `json:"actor_id"`
	Action       string    `gorm:"not null" json:"action"`
	TargetType   string    `gorm:"not null" json:"target_type"`
	TargetID     string    `gorm:"not null" json:"target_id"`
	MetadataJSON string    `gorm:"type:text" json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
}

func (Record) TableName() string {
	return "audit_events"
}

type Entry struct {
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Metadata   map[string]any
}

func (e Entry) toModel(id string) (Record, error) {
	rawMetadata := "{}"
	if e.Metadata != nil {
		payload, err := json.Marshal(e.Metadata)
		if err != nil {
			return Record{}, err
		}
		rawMetadata = string(payload)
	}

	return Record{
		ID:           id,
		ActorID:      e.ActorID,
		Action:       e.Action,
		TargetType:   e.TargetType,
		TargetID:     e.TargetID,
		MetadataJSON: rawMetadata,
	}, nil
}
