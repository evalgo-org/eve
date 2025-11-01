package semantic

// ConvertScheduleToISO8601 converts human-readable schedule to ISO 8601 duration
// Example: "every 4 hours" -> "PT4H"
func ConvertScheduleToISO8601(schedule string) string {
	switch schedule {
	case "every 1m", "every minute":
		return "PT1M"
	case "every 5m", "every 5 minutes":
		return "PT5M"
	case "every 15m", "every 15 minutes":
		return "PT15M"
	case "every 30m", "every 30 minutes":
		return "PT30M"
	case "every 1h", "every hour":
		return "PT1H"
	case "every 2h", "every 2 hours":
		return "PT2H"
	case "every 4h", "every 4 hours":
		return "PT4H"
	case "every 6h", "every 6 hours":
		return "PT6H"
	case "every 12h", "every 12 hours":
		return "PT12H"
	case "every 24h", "every day":
		return "PT24H"
	default:
		// Assume it's already a duration
		return schedule
	}
}

// ConvertISO8601ToSchedule converts ISO 8601 duration to human-readable schedule
// Example: "PT4H" -> "every 4h"
func ConvertISO8601ToSchedule(duration string) string {
	switch duration {
	case "PT1M":
		return "every 1m"
	case "PT5M":
		return "every 5m"
	case "PT15M":
		return "every 15m"
	case "PT30M":
		return "every 30m"
	case "PT1H":
		return "every 1h"
	case "PT2H":
		return "every 2h"
	case "PT4H":
		return "every 4h"
	case "PT6H":
		return "every 6h"
	case "PT12H":
		return "every 12h"
	case "PT24H":
		return "every 24h"
	default:
		return duration
	}
}
