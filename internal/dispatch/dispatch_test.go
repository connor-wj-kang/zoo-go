package dispatch

import "testing"

func TestGreedyCanBeWorseThanKM(t *testing.T) {
	candidates := []Candidate{
		{DemandID: "task-1", SupplyID: "student-1", Score: 95, DistanceKM: 0.3},
		{DemandID: "task-1", SupplyID: "student-2", Score: 90, DistanceKM: 0.5},
		{DemandID: "task-2", SupplyID: "student-1", Score: 89, DistanceKM: 0.7},
		{DemandID: "task-2", SupplyID: "student-2", Score: 10, DistanceKM: 4.8},
	}
	greedy := greedyFromCandidates(candidates)
	km := kmFromCandidates(candidates)
	if greedy.TotalScore >= km.TotalScore {
		t.Fatalf("expected KM to beat greedy, greedy=%+v km=%+v", greedy, km)
	}
	if km.Assigned != 2 || km.TotalScore != 179 {
		t.Fatalf("unexpected KM result: %+v", km)
	}
}

func TestBuildCandidatesRespectsRadius(t *testing.T) {
	demands := []Demand{{ID: "task-1", Latitude: 39.9998, Longitude: 116.3269}}
	supplies := []Supply{
		{ID: "near", Latitude: 40.005, Longitude: 116.330, Rating: 4.8, DoneCount: 12},
		{ID: "far", Latitude: 40.2, Longitude: 116.4, Rating: 5, DoneCount: 20},
	}
	candidates := BuildCandidates(demands, supplies, 5)
	if len(candidates) != 1 || candidates[0].SupplyID != "near" {
		t.Fatalf("candidates = %+v", candidates)
	}
	if candidates[0].Score <= 0 || candidates[0].Score > 100 {
		t.Fatalf("score = %.2f, want within 0..100", candidates[0].Score)
	}
}

func TestKMHandlesMoreStudentsThanTasks(t *testing.T) {
	candidates := []Candidate{
		{DemandID: "task-1", SupplyID: "student-1", Score: 70, DistanceKM: 2},
		{DemandID: "task-1", SupplyID: "student-2", Score: 90, DistanceKM: 0.5},
		{DemandID: "task-1", SupplyID: "student-3", Score: 80, DistanceKM: 1.2},
	}
	result := kmFromCandidates(candidates)
	if result.Assigned != 1 || result.Assignments[0].SupplyID != "student-2" {
		t.Fatalf("KM assignment = %+v", result)
	}
}

func TestKMHandlesMoreTasksThanStudents(t *testing.T) {
	candidates := []Candidate{
		{DemandID: "task-1", SupplyID: "student-1", Score: 95, DistanceKM: 0.3},
		{DemandID: "task-2", SupplyID: "student-1", Score: 90, DistanceKM: 0.5},
		{DemandID: "task-3", SupplyID: "student-1", Score: 85, DistanceKM: 0.8},
	}
	result := kmFromCandidates(candidates)
	if result.Assigned != 1 || result.Assignments[0].DemandID != "task-1" ||
		result.Assignments[0].SupplyID != "student-1" {
		t.Fatalf("KM assignment = %+v", result)
	}
}

func TestKMMaximizesCardinalityBeforeScore(t *testing.T) {
	candidates := []Candidate{
		{DemandID: "task-1", SupplyID: "student-1", Score: 99, DistanceKM: 0.1},
		{DemandID: "task-2", SupplyID: "student-2", Score: 20, DistanceKM: 4.9},
	}
	result := kmFromCandidates(candidates)
	if result.Assigned != 2 || result.TotalScore != 119 {
		t.Fatalf("KM assignment = %+v", result)
	}
}
