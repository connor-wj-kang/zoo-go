package httpapi

import (
	"context"
	"net/http"
	"time"

	"zoo-go/internal/domain"
)

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeData(w, s.store.Meta(s.now()))
}

func (s *Server) ownerHome(w http.ResponseWriter, r *http.Request) {
	home, err := s.store.OwnerHome(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, home)
}

func (s *Server) ownerProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.Profile(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, profile)
}

func (s *Server) ownerPets(w http.ResponseWriter, r *http.Request) {
	pets, err := s.store.Pets(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, pets)
}

func (s *Server) createPet(w http.ResponseWriter, r *http.Request) {
	var req domain.CreatePetRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	pet, err := s.store.CreatePet(currentUser(r).ID, req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: pet})
}

func (s *Server) ownerTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.OwnerTasks(currentUser(r).ID, query(r, "filter", "all"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, tasks)
}

func (s *Server) task(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	role := query(r, "role", "owner")
	view, err := s.store.Task(r.PathValue("id"), user.ID, role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) publishTask(w http.ResponseWriter, r *http.Request) {
	var req domain.PublishTaskRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.OwnerID = currentUser(r).ID
	view, err := s.store.PublishTask(req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: view})
}

func (s *Server) submitBid(w http.ResponseWriter, r *http.Request) {
	var req domain.SubmitBidRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.StudentID = currentUser(r).ID
	view, err := s.store.SubmitBid(r.PathValue("id"), req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) awardTask(w http.ResponseWriter, r *http.Request) {
	var req domain.AwardRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.OwnerID = currentUser(r).ID
	view, err := s.store.AwardTask(r.PathValue("id"), req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) reopenTask(w http.ResponseWriter, r *http.Request) {
	oldOrderID := s.taskOrderID(r.PathValue("id"), currentUser(r).ID)
	view, err := s.store.ReopenTask(r.PathValue("id"), currentUser(r).ID, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	s.closePaymentAsync(oldOrderID)
	writeData(w, view)
}

func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	oldOrderID := s.taskOrderID(r.PathValue("id"), currentUser(r).ID)
	view, err := s.store.CancelTask(r.PathValue("id"), currentUser(r).ID, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	s.closePaymentAsync(oldOrderID)
	writeData(w, view)
}

func (s *Server) taskOrderID(taskID, ownerID string) string {
	view, err := s.store.Task(taskID, ownerID, "owner")
	if err != nil {
		return ""
	}
	return view.OrderID
}

func (s *Server) closePaymentAsync(orderID string) {
	if s.payments == nil || orderID == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		s.payments.ClosePaymentForOrder(ctx, orderID)
	}()
}

func (s *Server) order(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	view, err := s.store.Order(r.PathValue("id"), user.ID, query(r, "role", "owner"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) checkIn(w http.ResponseWriter, r *http.Request) {
	view, err := s.store.CheckIn(r.PathValue("id"), currentUser(r).ID, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	var req domain.CompleteRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.StudentID = currentUser(r).ID
	view, err := s.store.Complete(r.PathValue("id"), req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	view, err := s.store.Confirm(r.PathValue("id"), currentUser(r).ID, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) rate(w http.ResponseWriter, r *http.Request) {
	var req domain.RateRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.OwnerID = currentUser(r).ID
	view, err := s.store.Rate(r.PathValue("id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) availableTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.AvailableTasks(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, tasks)
}

func (s *Server) updateStudentLocation(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateLocationRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	location, err := s.store.UpdateStudentLocation(currentUser(r).ID, req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, location)
}

func (s *Server) studentTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.StudentTasks(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, tasks)
}

func (s *Server) studentEarnings(w http.ResponseWriter, r *http.Request) {
	earnings, err := s.store.StudentEarnings(currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, earnings)
}

func (s *Server) studentProfile(w http.ResponseWriter, r *http.Request) {
	student, err := s.store.StudentProfileAt(currentUser(r).ID, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, student)
}

func (s *Server) submitStudentVerification(w http.ResponseWriter, r *http.Request) {
	var req domain.SubmitStudentVerificationRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	student, err := s.store.SubmitStudentVerification(currentUser(r).ID, req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, student)
}

func (s *Server) adminApproveStudent(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ApproveStudent(r.PathValue("id"), s.now()); err != nil {
		writeError(w, err)
		return
	}
	s.recordAdminAudit("STUDENT_VERIFICATION_APPROVED", "user", r.PathValue("id"), "兼容接口审核通过")
	writeData(w, map[string]bool{"ok": true})
}
