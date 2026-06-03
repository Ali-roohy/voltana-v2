package service_test

import (
	"context"
	"errors"
	"testing"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

func TestSettings_GetAutoCreates(t *testing.T) {
	svc := service.NewSettingsService(&mockSettingsRepo{}, newMockCarRepo())
	uid := uuid.New()

	st, err := svc.Get(context.Background(), uid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if st.UserID != uid {
		t.Error("settings must belong to the caller")
	}
	if st.PeakRate != 0 || st.MidRate != 0 || st.OffpeakRate != 0 || st.DefaultCarID != nil {
		t.Errorf("first-GET defaults should be zero rates / no default car, got %+v", st)
	}
}

func TestSettings_UpdateRates(t *testing.T) {
	svc := service.NewSettingsService(&mockSettingsRepo{}, newMockCarRepo())
	uid := uuid.New()

	st, err := svc.Update(context.Background(), uid, domain.SettingsInput{PeakRate: 12, MidRate: 6, OffpeakRate: 3})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if st.PeakRate != 12 || st.MidRate != 6 || st.OffpeakRate != 3 {
		t.Errorf("rates not persisted, got %+v", st)
	}
}

func TestSettings_RatesNonNegative(t *testing.T) {
	svc := service.NewSettingsService(&mockSettingsRepo{}, newMockCarRepo())
	cases := []domain.SettingsInput{
		{PeakRate: -1},
		{MidRate: -0.5},
		{OffpeakRate: -10},
	}
	for i, in := range cases {
		if _, err := svc.Update(context.Background(), uuid.New(), in); !errors.Is(err, service.ErrValidation) {
			t.Errorf("case %d: want ErrValidation, got %v", i, err)
		}
	}
}

func TestSettings_DefaultCarMustBeOwned(t *testing.T) {
	owner := uuid.New()
	carRepo := newMockCarRepo()
	car, err := carRepo.Create(context.Background(), owner, repository.CarInput{Name: "Mine"})
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewSettingsService(&mockSettingsRepo{}, carRepo)

	// a car the user does not own → 422 (ErrInvalidCarRef)
	if _, err := svc.Update(context.Background(), owner, domain.SettingsInput{DefaultCarID: ptr(uuid.New())}); !errors.Is(err, service.ErrInvalidCarRef) {
		t.Errorf("unowned default car: want ErrInvalidCarRef, got %v", err)
	}
	// the user's own car → ok
	st, err := svc.Update(context.Background(), owner, domain.SettingsInput{DefaultCarID: &car.ID})
	if err != nil {
		t.Fatalf("owned default car: %v", err)
	}
	if st.DefaultCarID == nil || *st.DefaultCarID != car.ID {
		t.Errorf("default_car_id not set, got %+v", st.DefaultCarID)
	}
}
