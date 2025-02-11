package domain

// Metadata represents the internal metadata type
type Metadata struct {
	ISRC         string            `json:"isrc"`
	ISWC         string            `json:"iswc"`
	BPM          float64           `json:"bpm"`
	Key          string            `json:"key"`
	Mood         string            `json:"mood"`
	Labels       []string          `json:"labels"`
	AITags       []string          `json:"aiTags"`
	Confidence   float64           `json:"confidence"`
	ModelVersion string            `json:"modelVersion"`
	CustomFields map[string]string `json:"customFields"`
}
