package domain

type DispatchTaskCandidate struct {
	ID        string  `json:"id"`
	Community string  `json:"community"`
	Latitude  float64 `json:"-"`
	Longitude float64 `json:"-"`
}

type DispatchStudentCandidate struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"-"`
	Longitude float64 `json:"-"`
	Rating    float64 `json:"rating"`
	DoneCount int     `json:"doneCount"`
}

type DispatchAssignmentView struct {
	TaskID        string  `json:"taskId"`
	TaskCommunity string  `json:"taskCommunity"`
	StudentID     string  `json:"studentId"`
	StudentName   string  `json:"studentName"`
	DistanceKM    float64 `json:"distanceKm"`
	Score         float64 `json:"score"`
}

type DispatchStrategyView struct {
	Strategy        string                   `json:"strategy"`
	Assigned        int                      `json:"assigned"`
	TotalScore      float64                  `json:"totalScore"`
	TotalDistanceKM float64                  `json:"totalDistanceKm"`
	Assignments     []DispatchAssignmentView `json:"assignments"`
}

type DispatchComparison struct {
	RadiusKM       float64              `json:"radiusKm"`
	TaskCount      int                  `json:"taskCount"`
	StudentCount   int                  `json:"studentCount"`
	CandidateEdges int                  `json:"candidateEdges"`
	Greedy         DispatchStrategyView `json:"greedy"`
	KM             DispatchStrategyView `json:"km"`
	ScoreGap       float64              `json:"scoreGap"`
	ReadOnly       bool                 `json:"readOnly"`
}
