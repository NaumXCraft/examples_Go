package service_test

import (
	"testing"
)

func TestAdd(t *testing.T) {
	s := New()

	todo, err := s.Add("Buy milk", "2 liters")
	if err != nil {
		t.Fatal(err)
	}
	if todo.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if todo.Title != "Buy milk" {
		t.Fatalf("expected title 'Buy milk', got %q", todo.Title)
	}
}

func TestAdd_EmptyTitle(t *testing.T) {
	s := New()

	_, err := s.Add("   ", "body")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestGet_NotFound(t *testing.T) {
	s := New()

	_, err := s.Get(999)
	if err == nil {
		t.Fatal("expected error for missing todo")
	}
}

func TestList_Filter(t *testing.T) {
	s := New()
	a, _ := s.Add("A", "")
	s.Add("B", "")
	s.Toggle(a.ID) // A теперь done = true

	doneOnly := true
	result := s.List(&doneOnly)

	if len(result) != 1 {
		t.Fatalf("expected 1 done todo, got %d", len(result))
	}
}

func TestUpdate(t *testing.T) {
	s := New()
	todo, _ := s.Add("Old title", "")

	newTitle := "New title"
	updated, err := s.Update(todo.ID, &newTitle, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "New title" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	todo, _ := s.Add("Temp", "")

	if err := s.Delete(todo.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(todo.ID); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestToggle(t *testing.T) {
	s := New()
	todo, _ := s.Add("T", "")

	toggled, _ := s.Toggle(todo.ID)
	if !toggled.Done {
		t.Fatal("expected done = true after first toggle")
	}

	toggledAgain, _ := s.Toggle(todo.ID)
	if toggledAgain.Done {
		t.Fatal("expected done = false after second toggle")
	}
}

func TestClear(t *testing.T) {
	s := New()
	s.Add("A", "")
	s.Add("B", "")

	s.Clear()

	if len(s.List(nil)) != 0 {
		t.Fatal("expected 0 todos after clear")
	}
}
