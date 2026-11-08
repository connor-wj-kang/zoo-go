package dispatch

import (
	"math"
	"sort"

	"zoo-go/internal/geo"
)

const (
	StrategyGreedy = "GREEDY"
	StrategyKM     = "KM"
)

type Demand struct {
	ID        string
	Latitude  float64
	Longitude float64
}

type Supply struct {
	ID        string
	Latitude  float64
	Longitude float64
	Rating    float64
	DoneCount int
}

type Candidate struct {
	DemandID   string
	SupplyID   string
	DistanceKM float64
	Score      float64
}

type Assignment struct {
	DemandID   string  `json:"demandId"`
	SupplyID   string  `json:"supplyId"`
	DistanceKM float64 `json:"distanceKm"`
	Score      float64 `json:"score"`
}

type Result struct {
	Strategy        string       `json:"strategy"`
	Assignments     []Assignment `json:"assignments"`
	Assigned        int          `json:"assigned"`
	TotalScore      float64      `json:"totalScore"`
	TotalDistanceKM float64      `json:"totalDistanceKm"`
}

func Compare(demands []Demand, supplies []Supply, radiusKM float64) (Result, Result, int) {
	if radiusKM <= 0 {
		radiusKM = geo.DefaultDispatchRadius
	}
	candidates := BuildCandidates(demands, supplies, radiusKM)
	return greedyFromCandidates(candidates), kmFromCandidates(candidates), len(candidates)
}

func BuildCandidates(demands []Demand, supplies []Supply, radiusKM float64) []Candidate {
	candidates := make([]Candidate, 0, len(demands)*len(supplies))
	for _, demand := range demands {
		demandPoint := geo.Point{Latitude: demand.Latitude, Longitude: demand.Longitude}
		if !geo.Valid(demandPoint) {
			continue
		}
		for _, supply := range supplies {
			supplyPoint := geo.Point{Latitude: supply.Latitude, Longitude: supply.Longitude}
			if !geo.Valid(supplyPoint) {
				continue
			}
			distance := geo.DistanceKM(demandPoint, supplyPoint)
			if distance > radiusKM {
				continue
			}
			candidates = append(candidates, Candidate{
				DemandID: demand.ID, SupplyID: supply.ID,
				DistanceKM: distance,
				Score:      EdgeScore(distance, radiusKM, supply.Rating, supply.DoneCount),
			})
		}
	}
	return candidates
}

func EdgeScore(distanceKM, radiusKM, rating float64, doneCount int) float64 {
	if radiusKM <= 0 || distanceKM < 0 || distanceKM > radiusKM {
		return 0
	}
	rating = math.Max(0, math.Min(5, rating))
	done := math.Max(0, math.Min(20, float64(doneCount)))
	proximity := 50 * (1 - distanceKM/radiusKM)
	quality := 12.5 + 12.5*(rating/5)
	experience := 15 * (done / 20)
	return proximity + quality + experience + 10
}

func greedyFromCandidates(candidates []Candidate) Result {
	ranked := append([]Candidate(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			if ranked[i].DistanceKM == ranked[j].DistanceKM {
				if ranked[i].DemandID == ranked[j].DemandID {
					return ranked[i].SupplyID < ranked[j].SupplyID
				}
				return ranked[i].DemandID < ranked[j].DemandID
			}
			return ranked[i].DistanceKM < ranked[j].DistanceKM
		}
		return ranked[i].Score > ranked[j].Score
	})
	usedDemands := map[string]bool{}
	usedSupplies := map[string]bool{}
	selected := []Candidate{}
	for _, candidate := range ranked {
		if usedDemands[candidate.DemandID] || usedSupplies[candidate.SupplyID] {
			continue
		}
		usedDemands[candidate.DemandID] = true
		usedSupplies[candidate.SupplyID] = true
		selected = append(selected, candidate)
	}
	sortCandidates(selected)
	return resultFromCandidates(StrategyGreedy, selected)
}

func kmFromCandidates(candidates []Candidate) Result {
	if len(candidates) == 0 {
		return resultFromCandidates(StrategyKM, nil)
	}
	demandIDs := uniqueIDs(candidates, true)
	supplyIDs := uniqueIDs(candidates, false)
	size := len(demandIDs)
	if len(supplyIDs) > size {
		size = len(supplyIDs)
	}
	byEdge := make(map[string]Candidate, len(candidates))
	for _, candidate := range candidates {
		byEdge[candidate.DemandID+"\x00"+candidate.SupplyID] = candidate
	}
	const (
		scoreScale     = int64(1000)
		maxScaledScore = int64(100 * scoreScale)
		matchBonus     = int64(1_000_000)
		dummyCost      = matchBonus + maxScaledScore
	)
	cost := make([][]int64, size)
	for i := range cost {
		cost[i] = make([]int64, size)
		for j := range cost[i] {
			cost[i][j] = dummyCost
		}
	}
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if i >= len(demandIDs) || j >= len(supplyIDs) {
				continue
			}
			candidate, ok := byEdge[demandIDs[i]+"\x00"+supplyIDs[j]]
			if !ok {
				continue
			}
			scaledScore := int64(math.Round(candidate.Score * float64(scoreScale)))
			scaledScore = max(0, min(maxScaledScore, scaledScore))
			cost[i][j] = dummyCost - matchBonus - scaledScore
		}
	}
	selected := []Candidate{}
	for demandIndex, supplyIndex := range hungarian(cost) {
		if supplyIndex < 0 || demandIndex >= len(demandIDs) || supplyIndex >= len(supplyIDs) {
			continue
		}
		if candidate, ok := byEdge[demandIDs[demandIndex]+"\x00"+supplyIDs[supplyIndex]]; ok {
			selected = append(selected, candidate)
		}
	}
	sortCandidates(selected)
	return resultFromCandidates(StrategyKM, selected)
}

func uniqueIDs(candidates []Candidate, demand bool) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, candidate := range candidates {
		id := candidate.SupplyID
		if demand {
			id = candidate.DemandID
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func sortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].DemandID == candidates[j].DemandID {
			return candidates[i].SupplyID < candidates[j].SupplyID
		}
		return candidates[i].DemandID < candidates[j].DemandID
	})
}

func resultFromCandidates(strategy string, candidates []Candidate) Result {
	result := Result{Strategy: strategy, Assignments: make([]Assignment, 0, len(candidates))}
	for _, candidate := range candidates {
		result.Assignments = append(result.Assignments, Assignment{
			DemandID: candidate.DemandID, SupplyID: candidate.SupplyID,
			DistanceKM: geo.RoundKM(candidate.DistanceKM), Score: math.Round(candidate.Score*10) / 10,
		})
		result.TotalScore += candidate.Score
		result.TotalDistanceKM += candidate.DistanceKM
	}
	result.Assigned = len(result.Assignments)
	result.TotalScore = math.Round(result.TotalScore*10) / 10
	result.TotalDistanceKM = math.Round(result.TotalDistanceKM*10) / 10
	return result
}

func hungarian(cost [][]int64) []int {
	n := len(cost)
	if n == 0 {
		return nil
	}
	const inf = int64(1 << 60)
	u := make([]int64, n+1)
	v := make([]int64, n+1)
	p := make([]int, n+1)
	way := make([]int, n+1)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]int64, n+1)
		used := make([]bool, n+1)
		for j := 0; j <= n; j++ {
			minv[j] = inf
		}
		for {
			used[j0] = true
			i0 := p[j0]
			delta := inf
			j1 := 0
			for j := 1; j <= n; j++ {
				if used[j] {
					continue
				}
				cur := cost[i0-1][j-1] - u[i0] - v[j]
				if cur < minv[j] {
					minv[j] = cur
					way[j] = j0
				}
				if minv[j] < delta {
					delta = minv[j]
					j1 = j
				}
			}
			for j := 0; j <= n; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}
	assignment := make([]int, n)
	for i := range assignment {
		assignment[i] = -1
	}
	for j := 1; j <= n; j++ {
		if p[j] > 0 {
			assignment[p[j]-1] = j - 1
		}
	}
	return assignment
}
