package service_test

import (
	"context"
	"errors"
	"testing"

	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

func boolPtr(b bool) *bool { return &b }

func TestAdminService_UpdateUser_RemoveSelfAdmin(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true

	svc := service.NewAdminService(repo)
	_, err := svc.UpdateUser(context.Background(), admin.ID, admin.ID, boolPtr(false), nil)
	if !errors.Is(err, service.ErrRemoveSelfAdmin) {
		t.Fatalf("want ErrRemoveSelfAdmin, got %v", err)
	}
}

func TestAdminService_UpdateUser_LastAdmin(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true
	other, _ := repo.Create(context.Background(), "other@test.com", "hash", nil, nil)

	svc := service.NewAdminService(repo)
	// admin tries to demote other when other is the only admin
	_, err := svc.UpdateUser(context.Background(), admin.ID, other.ID, boolPtr(false), nil)
	// admin is also in the map and is_admin=true, so count=1 for other... wait
	// Actually admin.IsAdmin=true and other.IsAdmin=false by default, so CountAdmins=1
	// demoting other (is_admin=false) when other is NOT admin: COALESCE keeps false, but guard runs because isAdmin=false
	// guard: callerID != targetID ✓, countAdmins=1 (only admin) → ErrLastAdmin
	if !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin, got %v", err)
	}
}

func TestAdminService_UpdateUser_GrantAdmin(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true
	other, _ := repo.Create(context.Background(), "other@test.com", "hash", nil, nil)

	svc := service.NewAdminService(repo)
	updated, err := svc.UpdateUser(context.Background(), admin.ID, other.ID, boolPtr(true), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.IsAdmin {
		t.Fatal("expected IsAdmin=true after granting admin")
	}
}

func TestAdminService_UpdateUser_VerifyEmail(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true
	other, _ := repo.Create(context.Background(), "other@test.com", "hash", nil, nil)

	svc := service.NewAdminService(repo)
	updated, err := svc.UpdateUser(context.Background(), admin.ID, other.ID, nil, boolPtr(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.IsEmailVerified {
		t.Fatal("expected IsEmailVerified=true")
	}
}

func TestAdminService_DeleteUser_Self(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true

	svc := service.NewAdminService(repo)
	err := svc.DeleteUser(context.Background(), admin.ID, admin.ID)
	if !errors.Is(err, service.ErrDeleteSelf) {
		t.Fatalf("want ErrDeleteSelf, got %v", err)
	}
}

func TestAdminService_DeleteUser_LastAdmin(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true
	other, _ := repo.Create(context.Background(), "other@test.com", "hash", nil, nil)
	_ = other

	svc := service.NewAdminService(repo)
	// deleting admin (the only admin) from another user's perspective
	err := svc.DeleteUser(context.Background(), other.ID, admin.ID)
	if !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin, got %v", err)
	}
}

func TestAdminService_DeleteUser_OK(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true
	other, _ := repo.Create(context.Background(), "other@test.com", "hash", nil, nil)

	svc := service.NewAdminService(repo)
	if err := svc.DeleteUser(context.Background(), admin.ID, other.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetUser(context.Background(), other.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatal("expected ErrNotFound after deletion")
	}
}

func TestAdminService_ListUsers(t *testing.T) {
	repo := newMockUserRepo()
	repo.Create(context.Background(), "a@test.com", "hash", nil, nil)
	repo.Create(context.Background(), "b@test.com", "hash", nil, nil)

	svc := service.NewAdminService(repo)
	users, total, err := svc.ListUsers(context.Background(), 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("want total=2, got %d", total)
	}
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %d", len(users))
	}
}

func TestAdminService_ListUsers_LimitClamped(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAdminService(repo)
	// limit > 100 should be clamped to 20 (doesn't error)
	_, _, err := svc.ListUsers(context.Background(), 9999, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminService_UpdateUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true

	svc := service.NewAdminService(repo)
	_, err := svc.UpdateUser(context.Background(), admin.ID, uuid.New(), boolPtr(true), nil)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// ── self-delete (TASK-0037 FEAT-5) ────────────────────────────────────────────

func TestSelfDelete_LastAdminRefused(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAdminService(repo)
	admin, _ := repo.Create(context.Background(), "admin@test.com", "hash", nil, nil)
	admin.IsAdmin = true

	err := svc.SelfDelete(context.Background(), admin.ID)
	if !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin for sole admin self-delete, got %v", err)
	}
	if _, err := repo.FindByID(context.Background(), admin.ID); err != nil {
		t.Fatal("admin must still exist after refused self-delete")
	}
}

func TestSelfDelete_NonAdminAndSecondAdminAllowed(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAdminService(repo)
	admin1, _ := repo.Create(context.Background(), "a1@test.com", "hash", nil, nil)
	admin1.IsAdmin = true
	admin2, _ := repo.Create(context.Background(), "a2@test.com", "hash", nil, nil)
	admin2.IsAdmin = true
	user, _ := repo.Create(context.Background(), "u@test.com", "hash", nil, nil)

	if err := svc.SelfDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("non-admin self-delete: %v", err)
	}
	if err := svc.SelfDelete(context.Background(), admin2.ID); err != nil {
		t.Fatalf("second-admin self-delete (another admin remains): %v", err)
	}
	if err := svc.SelfDelete(context.Background(), admin1.ID); !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin once only one admin remains, got %v", err)
	}
}
