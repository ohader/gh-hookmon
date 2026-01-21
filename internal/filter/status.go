package filter

// IsFailed checks if a delivery represents a failed webhook delivery
// Failed deliveries are:
// - HTTP status code >= 400 (4xx and 5xx errors)
// - Status code 0 (no response/delivery failed)
func IsFailed(statusCode int) bool {
	return statusCode == 0 || statusCode >= 400
}

// IsSuccessful checks if a delivery represents a successful webhook delivery
// Successful deliveries are HTTP status code 200-399
func IsSuccessful(statusCode int) bool {
	return statusCode >= 200 && statusCode < 400
}

// MatchesStatus checks if a delivery matches the given status filter
// filterType can be: "failed", "successful", "all" (or empty for no filter)
// This provides extensibility for future --status flag
func MatchesStatus(statusCode int, filterType string) bool {
	switch filterType {
	case "failed":
		return IsFailed(statusCode)
	case "successful":
		return IsSuccessful(statusCode)
	case "all", "":
		return true
	default:
		return true
	}
}
