package service

import (
	"context"
	"errors"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
)

var (
	ErrInvalidOTPDeliveryMethod = errors.New("otp_delivery_method must be 'deeplink' or 'contact_share'")
	ErrInvalidDefaultRates      = errors.New("default rates must be non-negative")
)

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
	rates, err := s.repo.GetDefaultRates(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.SystemSettings{
		OTPDeliveryMethod:  method,
		DefaultPeakRate:    rates.Peak,
		DefaultMidRate:     rates.Mid,
		DefaultOffpeakRate: rates.Offpeak,
	}, nil
}

// SetDefaultRates stores the admin default rates copied into NEW users'
// settings. Existing users' rates and sessions are never touched.
func (s *SystemSettingsService) SetDefaultRates(ctx context.Context, peak, mid, offpeak float64) error {
	if peak < 0 || mid < 0 || offpeak < 0 {
		return ErrInvalidDefaultRates
	}
	return s.repo.SetDefaultRates(ctx, repository.Rates{Peak: peak, Mid: mid, Offpeak: offpeak})
}

func (s *SystemSettingsService) SetOTPDeliveryMethod(ctx context.Context, method string) error {
	if method != "deeplink" && method != "contact_share" {
		return ErrInvalidOTPDeliveryMethod
	}
	return s.repo.SetOTPDeliveryMethod(ctx, method)
}
