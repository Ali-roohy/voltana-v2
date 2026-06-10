package service_test

import (
	"context"
	"errors"
	"testing"

	"voltana-api/internal/service"
)

type mockSystemSettingsRepo struct {
	method string
}

func (m *mockSystemSettingsRepo) GetOTPDeliveryMethod(_ context.Context) (string, error) {
	if m.method == "" {
		return "deeplink", nil
	}
	return m.method, nil
}

func (m *mockSystemSettingsRepo) SetOTPDeliveryMethod(_ context.Context, method string) error {
	m.method = method
	return nil
}

func TestSystemSettingsService_GetSettings_Default(t *testing.T) {
	svc := service.NewSystemSettingsService(&mockSystemSettingsRepo{})
	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.OTPDeliveryMethod != "deeplink" {
		t.Errorf("want deeplink default, got %s", settings.OTPDeliveryMethod)
	}
}

func TestSystemSettingsService_GetSettings_ContactShare(t *testing.T) {
	svc := service.NewSystemSettingsService(&mockSystemSettingsRepo{method: "contact_share"})
	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.OTPDeliveryMethod != "contact_share" {
		t.Errorf("want contact_share, got %s", settings.OTPDeliveryMethod)
	}
}

func TestSystemSettingsService_SetOTPDeliveryMethod_Valid(t *testing.T) {
	repo := &mockSystemSettingsRepo{}
	svc := service.NewSystemSettingsService(repo)
	for _, m := range []string{"deeplink", "contact_share"} {
		if err := svc.SetOTPDeliveryMethod(context.Background(), m); err != nil {
			t.Errorf("SetOTPDeliveryMethod(%q): unexpected error: %v", m, err)
		}
		if repo.method != m {
			t.Errorf("want %q stored, got %q", m, repo.method)
		}
	}
}

func TestSystemSettingsService_SetOTPDeliveryMethod_Invalid(t *testing.T) {
	svc := service.NewSystemSettingsService(&mockSystemSettingsRepo{})
	err := svc.SetOTPDeliveryMethod(context.Background(), "sms")
	if !errors.Is(err, service.ErrInvalidOTPDeliveryMethod) {
		t.Errorf("want ErrInvalidOTPDeliveryMethod, got %v", err)
	}
}
