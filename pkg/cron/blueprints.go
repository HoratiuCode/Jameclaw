package cron

import (
	"fmt"
	"regexp"
	"strings"
)

type BlueprintSlot struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Label    string   `json:"label"`
	Default  string   `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"`
	Optional bool     `json:"optional"`
	Strict   bool     `json:"strict"`
	Help     string   `json:"help,omitempty"`
}

type AutomationBlueprint struct {
	Key              string          `json:"key"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Category         string          `json:"category"`
	ScheduleTemplate string          `json:"-"`
	PromptTemplate   string          `json:"-"`
	Fields           []BlueprintSlot `json:"fields"`
	Tags             []string        `json:"tags"`
}

type BlueprintJob struct {
	Name     string
	Schedule CronSchedule
	Prompt   string
	Deliver  bool
}

var weekdayPresets = map[string]string{
	"everyday": "*",
	"weekdays": "1-5",
	"weekends": "0,6",
}

var dayNames = map[string]string{
	"sunday": "0", "sun": "0",
	"monday": "1", "mon": "1",
	"tuesday": "2", "tue": "2",
	"wednesday": "3", "wed": "3",
	"thursday": "4", "thu": "4",
	"friday": "5", "fri": "5",
	"saturday": "6", "sat": "6",
}

var AutomationBlueprints = []AutomationBlueprint{
	{
		Key:              "morning-brief",
		Title:            "Morning briefing",
		Description:      "A short daily briefing with today's calendar, weather, and urgent items.",
		Category:         "daily",
		ScheduleTemplate: "{minute} {hour} * * *",
		PromptTemplate:   "Produce a concise morning briefing for the user: today's calendar events, the local weather, and any urgent items. Keep it short and scannable. If no data sources are connected, give a brief good-morning with the date and offer to connect calendar/email.",
		Fields:           []BlueprintSlot{timeSlot("08:00"), deliverSlot()},
		Tags:             []string{"daily", "briefing"},
	},
	{
		Key:              "important-mail",
		Title:            "Important-mail monitor",
		Description:      "Check your inbox periodically and ping only about mail that needs attention.",
		Category:         "email",
		ScheduleTemplate: "*/{interval_min} * * * *",
		PromptTemplate:   "Check the user's inbox for new messages since the last run. Surface ONLY mail matching: {criteria}. Score candidates by urgency and deliver only what clears the bar; if nothing does, respond with [SILENT]. Requires a connected mail source; if none is configured, explain how to connect one and stop.",
		Fields: []BlueprintSlot{
			{Name: "interval_min", Type: "enum", Label: "How often?", Default: "30", Options: []string{"15", "30", "60"}, Strict: true, Help: "minutes between checks"},
			{Name: "criteria", Type: "text", Label: "Only notify me if the mail...", Default: "needs a reply today, is from my manager or family, or mentions a deadline", Strict: false},
			deliverSlot(),
		},
		Tags: []string{"email", "monitor"},
	},
	{
		Key:              "weekly-review",
		Title:            "Weekly review",
		Description:      "A weekly recap of what got done, what's still open, and what's coming up.",
		Category:         "weekly",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Produce a weekly review for the user: what was accomplished this week, still-open items, and next week's calendar. Pull from connected sources. Keep it tight.",
		Fields: []BlueprintSlot{
			timeSlot("18:00"),
			{Name: "day", Type: "enum", Label: "Which day?", Default: "sunday", Options: []string{"sunday", "monday", "friday", "saturday"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"weekly", "review"},
	},
	{
		Key:              "custom-reminder",
		Title:            "Custom reminder",
		Description:      "A recurring reminder in your own words, on your schedule.",
		Category:         "general",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Remind the user: {what}",
		Fields: []BlueprintSlot{
			{Name: "what", Type: "text", Label: "Remind me to...", Default: "take a break and stretch", Strict: false},
			timeSlot("14:00"),
			{Name: "recurrence", Type: "weekdays", Label: "Repeat on", Default: "everyday", Options: []string{"everyday", "weekdays", "weekends"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"reminder"},
	},
	{
		Key:              "news-digest",
		Title:            "Topic news digest",
		Description:      "A recurring digest on a topic you care about, deduped against previous runs.",
		Category:         "general",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Search the web for new and noteworthy items about: {topic}. Dedupe against what you sent in previous runs; only include genuinely new developments. Deliver a tight digest of at most {count} bullets, each one line with a link. If nothing new since last run, respond with [SILENT].",
		Fields: []BlueprintSlot{
			{Name: "topic", Type: "text", Label: "What topic?", Default: "AI and technology", Strict: false, Help: "a subject, product, person, or search phrase"},
			timeSlot("18:00"),
			{Name: "recurrence", Type: "weekdays", Label: "Repeat on", Default: "weekdays", Options: []string{"everyday", "weekdays", "weekends"}, Strict: true},
			{Name: "count", Type: "enum", Label: "How many bullets?", Default: "5", Options: []string{"3", "5", "8"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"digest", "research"},
	},
	{
		Key:              "habit-checkin",
		Title:            "Habit check-in",
		Description:      "A recurring nudge to keep a habit on track.",
		Category:         "general",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Nudge the user about their habit: {habit}. Ask whether they did it today, keep it warm and non-judgmental, and offer a one-line word of encouragement. One short message.",
		Fields: []BlueprintSlot{
			{Name: "habit", Type: "text", Label: "Which habit?", Default: "20 minutes of reading", Strict: false},
			timeSlot("20:00"),
			{Name: "recurrence", Type: "weekdays", Label: "Repeat on", Default: "everyday", Options: []string{"everyday", "weekdays", "weekends"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"habit", "wellbeing"},
	},
	{
		Key:              "hydration-move",
		Title:            "Hydration & movement nudge",
		Description:      "A periodic weekday nudge to drink water, stand up, and stretch.",
		Category:         "general",
		ScheduleTemplate: "0 {start_hour}-{end_hour}/{interval_hours} * * 1-5",
		PromptTemplate:   "Send the user a brief, friendly nudge to drink some water, stand up, and stretch for a moment. Vary the wording each time so it doesn't feel robotic. One short line.",
		Fields: []BlueprintSlot{
			{Name: "interval_hours", Type: "enum", Label: "How often?", Default: "1", Options: []string{"1", "2", "3"}, Strict: true, Help: "hours between nudges"},
			{Name: "start_hour", Type: "enum", Label: "Start hour", Default: "9", Options: []string{"7", "8", "9", "10"}, Strict: true},
			{Name: "end_hour", Type: "enum", Label: "End hour", Default: "17", Options: []string{"16", "17", "18", "19"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"wellbeing", "focus"},
	},
	{
		Key:              "meal-plan",
		Title:            "Weekly meal plan",
		Description:      "A weekly meal plan plus a consolidated grocery list.",
		Category:         "weekly",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Build the user a meal plan for the coming week: {meals} per day, suited to a {diet} diet and roughly {effort} cooking effort. Include a consolidated grocery list grouped by aisle. Keep it simple and skimmable.",
		Fields: []BlueprintSlot{
			{Name: "diet", Type: "enum", Label: "Diet?", Default: "no restrictions", Options: []string{"no restrictions", "vegetarian", "vegan", "high-protein", "low-carb"}, Strict: true},
			{Name: "meals", Type: "enum", Label: "Meals per day?", Default: "dinner only", Options: []string{"dinner only", "lunch and dinner", "all three"}, Strict: true},
			{Name: "effort", Type: "enum", Label: "Cooking effort?", Default: "quick", Options: []string{"quick", "medium", "ambitious"}, Strict: true},
			timeSlot("17:00"),
			{Name: "day", Type: "enum", Label: "Which day?", Default: "sunday", Options: []string{"sunday", "monday", "friday", "saturday"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"weekly", "food"},
	},
	{
		Key:              "learn-daily",
		Title:            "Daily learning drip",
		Description:      "One bite-sized lesson a day on a topic you want to learn.",
		Category:         "daily",
		ScheduleTemplate: "{minute} {hour} * * {dow}",
		PromptTemplate:   "Teach the user one bite-sized lesson about: {topic}. Build on earlier lessons so it progresses rather than repeating. Keep it to a couple of short paragraphs with one concrete example, and end with a single question to check understanding.",
		Fields: []BlueprintSlot{
			{Name: "topic", Type: "text", Label: "Learn about...", Default: "Spanish vocabulary", Strict: false},
			timeSlot("08:30"),
			{Name: "recurrence", Type: "weekdays", Label: "Repeat on", Default: "weekdays", Options: []string{"everyday", "weekdays", "weekends"}, Strict: true},
			deliverSlot(),
		},
		Tags: []string{"learning", "daily"},
	},
}

func timeSlot(defaultValue string) BlueprintSlot {
	return BlueprintSlot{Name: "time", Type: "time", Label: "What time?", Default: defaultValue, Strict: true, Help: "24h local time, e.g. 08:00"}
}

func deliverSlot() BlueprintSlot {
	return BlueprintSlot{Name: "deliver", Type: "enum", Label: "Where to deliver?", Default: "local", Options: []string{"local", "origin"}, Strict: true, Help: "local = run and save result in JameClaw; origin is used only when created from an active chat"}
}

func GetAutomationBlueprint(key string) (AutomationBlueprint, bool) {
	for _, blueprint := range AutomationBlueprints {
		if blueprint.Key == key {
			return blueprint, true
		}
	}
	return AutomationBlueprint{}, false
}

func FillAutomationBlueprint(key string, values map[string]string) (BlueprintJob, error) {
	blueprint, ok := GetAutomationBlueprint(key)
	if !ok {
		return BlueprintJob{}, fmt.Errorf("unknown automation blueprint %q", key)
	}
	resolved, err := resolveBlueprintValues(blueprint, values)
	if err != nil {
		return BlueprintJob{}, err
	}
	scheduleExpr, err := fillScheduleTemplate(blueprint.ScheduleTemplate, resolved)
	if err != nil {
		return BlueprintJob{}, err
	}
	prompt := fillTemplate(blueprint.PromptTemplate, resolved)
	return BlueprintJob{
		Name:     blueprint.Title,
		Schedule: CronSchedule{Kind: "cron", Expr: scheduleExpr},
		Prompt:   prompt,
		Deliver:  resolved["deliver"] == "origin",
	}, nil
}

func resolveBlueprintValues(blueprint AutomationBlueprint, values map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(blueprint.Fields)+4)
	for _, slot := range blueprint.Fields {
		value := strings.TrimSpace(values[slot.Name])
		if value == "" {
			value = slot.Default
		}
		if value == "" && !slot.Optional {
			return nil, fmt.Errorf("%s is required", slot.Label)
		}
		if slot.Strict && len(slot.Options) > 0 && value != "" && !containsString(slot.Options, value) {
			return nil, fmt.Errorf("%s must be one of: %s", slot.Label, strings.Join(slot.Options, ", "))
		}
		if slot.Type == "time" {
			hour, minute, err := parseBlueprintTime(value)
			if err != nil {
				return nil, fmt.Errorf("%s must be HH:MM", slot.Label)
			}
			resolved[slot.Name] = value
			resolved["hour"] = hour
			resolved["minute"] = minute
			continue
		}
		if slot.Type == "weekdays" {
			dow, ok := weekdayPresets[value]
			if !ok {
				return nil, fmt.Errorf("%s must be one of: everyday, weekdays, weekends", slot.Label)
			}
			resolved[slot.Name] = value
			resolved["dow"] = dow
			continue
		}
		if slot.Name == "day" {
			dow, ok := dayNames[strings.ToLower(value)]
			if !ok {
				return nil, fmt.Errorf("%s must be a weekday", slot.Label)
			}
			resolved[slot.Name] = value
			resolved["dow"] = dow
			continue
		}
		resolved[slot.Name] = value
	}
	if _, ok := resolved["deliver"]; !ok {
		resolved["deliver"] = "local"
	}
	return resolved, nil
}

func parseBlueprintTime(value string) (string, string, error) {
	if !regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(value) {
		return "", "", fmt.Errorf("invalid time")
	}
	parts := strings.Split(value, ":")
	hour := strings.TrimLeft(parts[0], "0")
	minute := strings.TrimLeft(parts[1], "0")
	if hour == "" {
		hour = "0"
	}
	if minute == "" {
		minute = "0"
	}
	var h, m int
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil {
		return "", "", err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return "", "", err
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return "", "", fmt.Errorf("invalid time")
	}
	return hour, minute, nil
}

func fillScheduleTemplate(template string, values map[string]string) (string, error) {
	output := fillTemplate(template, values)
	if strings.Contains(output, "{") || strings.Contains(output, "}") {
		return "", fmt.Errorf("blueprint schedule has unresolved values")
	}
	return output, nil
}

func fillTemplate(template string, values map[string]string) string {
	output := template
	for key, value := range values {
		output = strings.ReplaceAll(output, "{"+key+"}", value)
	}
	return output
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
