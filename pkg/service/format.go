package service

// FormatState converts an API state string to a human-readable label.
func FormatState(state string) string {
	switch state {
	case "PUBLISHED":
		return "Published"
	case "PENDING_REVIEW":
		return "Pending Review"
	case "STAGED":
		return "Staged"
	case "PUBLISHED_TO_TESTERS":
		return "Published to Testers"
	case "REJECTED":
		return "Rejected"
	case "CANCELLED":
		return "Cancelled"
	case "ITEM_STATE_UNSPECIFIED":
		return "Unknown"
	default:
		return state
	}
}
