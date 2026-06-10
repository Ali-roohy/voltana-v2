package service

import (
	"context"
	"errors"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
)

var ErrInvalidOTPDeliveryMethod = errors.New("otp_delivery_method must be 'deeplink' or 'contact_share'")

// SystemSettingsService manages operator-level application settings.
type SystemSettingsService struct {
	repo repository.SystemSettingsRepository
}

func NewSystemSettingsService(repo repository.SystemSettingsRepository) *SystemSettingsService {
	return &SystemSettingsService{repo: repo}
}

func (s *SystemSettingsService) GetSettings(ctx context.Context) (*domain.SystemSettings, error) {
	method, err := s.repo.GetOTPDeliveryMethod(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.SystemSettings{OTPDeliveryMethod: method}, nil
}

func (s *SystemSettingsService) SetOTPDeliveryMethod(ctx context.Context, method string) error {
	if method != "deeplink" && method != "contact_share" {
		return ErrInvalidOTPDeliveryMethod
	}
	return s.repo.SetOTPDeliveryMethod(ctx, method)
}
