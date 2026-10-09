package daily_action

// GenerationStatus represents the status of a daily action generation run.
type GenerationStatus string

const (
	GenerationStatusStarted    GenerationStatus = "started"
	GenerationStatusGenerating GenerationStatus = "generating"
	GenerationStatusReady      GenerationStatus = "ready"
	GenerationStatusStale      GenerationStatus = "stale"
)

// ActionType represents the type of daily action.
type ActionType string

const (
	ActionTypeOutreach    ActionType = "outreach"
	ActionTypeFollowup    ActionType = "followup"
	ActionTypeRespond     ActionType = "respond"
	ActionTypeMeetingPrep ActionType = "meeting_prep"
	ActionTypeEnrich      ActionType = "enrich"
)

// ActionStatus represents the lifecycle status of a daily action.
type ActionStatus string

const (
	ActionStatusSuggested  ActionStatus = "suggested"
	ActionStatusInProgress ActionStatus = "in_progress"
	ActionStatusCompleted  ActionStatus = "completed"
	ActionStatusSkipped    ActionStatus = "skipped"
	ActionStatusSnoozed    ActionStatus = "snoozed"
	ActionStatusDeferred   ActionStatus = "deferred"
)

// CategoryID represents the category grouping for actions.
type CategoryID string

const (
	CategoryReplied     CategoryID = "replied"
	CategoryFollowup    CategoryID = "followup"
	CategoryNewOutreach CategoryID = "new_outreach"
	CategoryMeetingPrep CategoryID = "meeting_prep"
	CategoryEnrichment  CategoryID = "enrichment"
)

// Transition represents the state transition commands for an action.
type Transition string

const (
	TransitionMarkSent      Transition = "mark_sent"
	TransitionSkip          Transition = "skip"
	TransitionSnooze        Transition = "snooze"
	TransitionSnoozeCustom  Transition = "snooze_custom"
	TransitionMarkCompleted Transition = "mark_completed"
	TransitionReopen        Transition = "reopen"
)

// ValidTransitions maps each action status to its allowed transitions.
var ValidTransitions = map[ActionStatus][]Transition{
	ActionStatusSuggested:  {TransitionMarkSent, TransitionSkip, TransitionSnooze, TransitionSnoozeCustom, TransitionMarkCompleted},
	ActionStatusInProgress: {TransitionMarkCompleted},
	ActionStatusSkipped:    {TransitionReopen},
	ActionStatusSnoozed:    {TransitionReopen},
	ActionStatusCompleted:  {TransitionReopen},
	ActionStatusDeferred:   {TransitionReopen},
}

// IsValidTransition checks whether a transition is allowed from the given status.
func IsValidTransition(from ActionStatus, t Transition) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == t {
			return true
		}
	}
	return false
}
