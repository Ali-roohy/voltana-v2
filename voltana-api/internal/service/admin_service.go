package service

import (
	"context"
	"errors"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrRemoveSelfAdmin = errors.New("cannot remove own admin")
	ErrLastAdmin       = errors.New("cannot remove the last admin")
	ErrDeleteSelf      = errors.New("cannot delete own account from admin panel")
)

type AdminService struct {
	users repository.UserRepository
}

func NewAdminService(users repository.UserRepository) *AdminService {
	return &AdminService{users: users}
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.users.ListAll(ctx, limit, offset)
}

func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *AdminService) UpdateUser(ctx context.Context, callerID, targetID uuid.UUID, isAdmin *bool, isEmailVerified *bool) (*domain.User, error) {
	if isAdmin != nil && !*isAdmin {
		if callerID == targetID {
			return nil, ErrRemoveSelfAdmin
		}
		count, err := s.users.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, ErrLastAdmin
		}
	}
	u, err := s.users.AdminUpdate(ctx, targetID, isAdmin, isEmailVerified)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (s *AdminService) DeleteUser(ctx context.Context, callerID, targetID uuid.UUID) error {
	if callerID == targetID {
		return ErrDeleteSelf
	}
	target, err := s.users.FindByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.IsAdmin {
		count, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}
	return s.users.Delete(ctx, targetID)
}

// SelfDelete removes the authenticated user's own account (TASK-0037 FEAT-5).
// All owned rows cascade via FK (cars, sessions, snapshots, settings). The
// permanent first admin is protected by the last-admin guard: an admin can
// only self-delete when another admin exists.
func (s *AdminService) SelfDelete(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.IsAdmin {
		count, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}
	return s.users.Delete(ctx, userID)
}
