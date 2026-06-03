package service

import "voltana-api/internal/domain"

// recommendFor returns chemistry-aware battery-care advice. Chemistry may be nil
// (the car has no linked ev_model, or the model has no chemistry recorded) — in
// that case conservative generic advice is returned. NCA is treated like NMC.
func recommendFor(chemistry *string) domain.BatteryRecommendation {
	rec := domain.BatteryRecommendation{Chemistry: chemistry}

	switch normalizeChemistry(chemistry) {
	case "LFP":
		rec.ChargeCeiling = 100
		rec.Tips = []string{
			"LFP packs are happy charged to 100% — do it regularly.",
			"Charge to full at least weekly so the BMS can calibrate the state-of-charge reading.",
			"Avoid leaving the car at a very low charge for long periods.",
		}
	case "NMC", "NCA":
		rec.ChargeCeiling = 80
		rec.Tips = []string{
			"For daily use, charge to about 80% to slow calendar ageing.",
			"Charge to 100% only right before a long trip.",
			"Avoid regularly draining below 10–20%.",
		}
	default: // unknown / nil chemistry — conservative generic advice
		rec.ChargeCeiling = 80
		rec.Tips = []string{
			"Battery chemistry is unknown — a daily ceiling of about 80% is a safe default.",
			"Avoid frequent deep discharges and prolonged storage at full charge.",
			"Link this car to its EV model to get chemistry-specific advice.",
		}
	}
	return rec
}

func normalizeChemistry(chemistry *string) string {
	if chemistry == nil {
		return ""
	}
	return *chemistry
}
