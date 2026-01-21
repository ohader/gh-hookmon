package filter

import "time"

// InRange checks if the delivered time is within the specified date range
func InRange(deliveredAt time.Time, since, until *time.Time) bool {
	if since != nil && deliveredAt.Before(*since) {
		return false
	}
	if until != nil && deliveredAt.After(*until) {
		return false
	}
	return true
}
