package domain

// SystemSettings holds operator-configurable application settings persisted in
// the system_settings table. Add new fields here as more settings are introduced.
type SystemSettings struct {
	OTPDeliveryMethod string // "deeplink" | "contact_share"
}
