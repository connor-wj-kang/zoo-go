package httpapi

import (
	"net/http"

	"zoo-go/internal/dispatch"
	"zoo-go/internal/domain"
	"zoo-go/internal/geo"
)

func (s *Server) adminDispatchCompare(w http.ResponseWriter, r *http.Request) {
	tasks, students, err := s.store.AdminDispatchInputs(s.now(), limitQuery(r, 50))
	if err != nil {
		writeError(w, err)
		return
	}
	demands := make([]dispatch.Demand, 0, len(tasks))
	taskCommunities := make(map[string]string, len(tasks))
	for _, task := range tasks {
		demands = append(demands, dispatch.Demand{
			ID: task.ID, Latitude: task.Latitude, Longitude: task.Longitude,
		})
		taskCommunities[task.ID] = task.Community
	}
	supplies := make([]dispatch.Supply, 0, len(students))
	studentNames := make(map[string]string, len(students))
	for _, student := range students {
		supplies = append(supplies, dispatch.Supply{
			ID: student.ID, Latitude: student.Latitude, Longitude: student.Longitude,
			Rating: student.Rating, DoneCount: student.DoneCount,
		})
		studentNames[student.ID] = student.Name
	}
	greedy, km, candidateEdges := dispatch.Compare(demands, supplies, geo.DefaultDispatchRadius)
	response := domain.DispatchComparison{
		RadiusKM: geo.DefaultDispatchRadius, TaskCount: len(tasks), StudentCount: len(students),
		CandidateEdges: candidateEdges, ReadOnly: true,
		Greedy: dispatchStrategyView(greedy, taskCommunities, studentNames),
		KM:     dispatchStrategyView(km, taskCommunities, studentNames),
	}
	response.ScoreGap = km.TotalScore - greedy.TotalScore
	writeData(w, response)
}

func dispatchStrategyView(result dispatch.Result, taskCommunities, studentNames map[string]string) domain.DispatchStrategyView {
	view := domain.DispatchStrategyView{
		Strategy: result.Strategy, Assigned: result.Assigned, TotalScore: result.TotalScore,
		TotalDistanceKM: result.TotalDistanceKM, Assignments: make([]domain.DispatchAssignmentView, 0, len(result.Assignments)),
	}
	for _, assignment := range result.Assignments {
		view.Assignments = append(view.Assignments, domain.DispatchAssignmentView{
			TaskID: assignment.DemandID, TaskCommunity: taskCommunities[assignment.DemandID],
			StudentID: assignment.SupplyID, StudentName: studentNames[assignment.SupplyID],
			DistanceKM: assignment.DistanceKM, Score: assignment.Score,
		})
	}
	return view
}
