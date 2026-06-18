package service_test

import (
	"testing"
	"todo-api/service"
)

func newSvc() *service.TodoService { return service.New(0) }

func TestAdd_OK(t *testing.T) {
	s := newSvc()
	got, err := s.Add(service.CreateInput{Title: "Task 1", Body: "body"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 || got.Title != "Task 1" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestAdd_EmptyTitle(t *testing.T) {
	s := newSvc()
	_, err := s.Add(service.CreateInput{Title: "   "})
	if err != service.ErrTitleEmpty {
		t.Fatalf("expected ErrTitleEmpty, got %v", err)
	}
}

func TestAdd_Limit(t *testing.T) {
	s := service.New(1)
	s.Add(service.CreateInput{Title: "A"})
	_, err := s.Add(service.CreateInput{Title: "B"})
	if err != service.ErrLimitReached {
		t.Fatalf("expected ErrLimitReached, got %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := newSvc()
	_, err := s.Get(99)
	if err != service.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_Filter(t *testing.T) {
	s := newSvc()
	a, _ := s.Add(service.CreateInput{Title: "A"})
	s.Add(service.CreateInput{Title: "B"})
	s.Toggle(a.ID)

	trueVal := true
	done := s.List(&trueVal)
	if len(done) != 1 {
		t.Fatalf("expected 1 done, got %d", len(done))
	}
}

func TestUpdate(t *testing.T) {
	s := newSvc()
	a, _ := s.Add(service.CreateInput{Title: "Old"})
	newTitle := "New"
	updated, err := s.Update(a.ID, service.UpdateInput{Title: &newTitle})
	if err != nil || updated.Title != "New" {
		t.Fatalf("update failed: %v %+v", err, updated)
	}
}

func TestDelete(t *testing.T) {
	s := newSvc()
	a, _ := s.Add(service.CreateInput{Title: "X"})
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(a.ID); err != service.ErrNotFound {
		t.Fatalf("expected ErrNotFound on second delete")
	}
}

func TestToggle(t *testing.T) {
	s := newSvc()
	a, _ := s.Add(service.CreateInput{Title: "T"})
	tog, _ := s.Toggle(a.ID)
	if !tog.Done {
		t.Fatal("expected done=true")
	}
	tog2, _ := s.Toggle(a.ID)
	if tog2.Done {
		t.Fatal("expected done=false after second toggle")
	}
}

func TestClear(t *testing.T) {
	s := newSvc()
	s.Add(service.CreateInput{Title: "A"})
	s.Add(service.CreateInput{Title: "B"})
	s.Clear()
	if len(s.List(nil)) != 0 {
		t.Fatal("expected empty after clear")
	}
}
