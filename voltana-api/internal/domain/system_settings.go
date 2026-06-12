package domain

// SystemSettings holds operator-configurable application settings persisted in
// the system_settings table. Add new fields here as more settings are introduced.
type SystemSettings struct {
	OTPDeliveryMethod string // "deeplink" | "contact_share"
	// Admin default electricity rates, copied into NEW users' settings at
	// account/settings-row creation (TASK-0037 FEAT-6).
	DefaultPeakRate    float64
	DefaultMidRate     float64
	DefaultOffpeakRate float64
}
