package timeutil

import "time"

// Beijing is GMT+8. All persisted timestamps use this location.
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func Parse(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, Beijing); err == nil {
		return t, nil
	}
	return time.ParseInLocation(time.RFC3339, s, Beijing)
}

func CivilDate(t time.Time) (y int, m time.Month, d int) {
	return t.In(Beijing).Date()
}
