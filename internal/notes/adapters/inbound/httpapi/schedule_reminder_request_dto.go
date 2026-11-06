package httpapi

type ScheduleReminderRequestDTO struct {
	RemindAt string `json:"remindAt"`
}

var scheduleReminderRequestFields = []string{remindAtField}
