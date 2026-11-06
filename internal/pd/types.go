package pd

import "time"

type Scope int

const (
	ScopeMine Scope = iota
	ScopeTeam
	ScopeAll
)

func (s Scope) String() string {
	switch s {
	case ScopeMine:
		return "Mine"
	case ScopeTeam:
		return "My Teams"
	case ScopeAll:
		return "All"
	default:
		return "Unknown"
	}
}

type Incident struct {
	ID           string
	Number       uint
	Title        string
	Status       string // triggered, acknowledged, resolved
	Urgency      string // high, low
	ServiceName  string
	AssignedTo   string
	CreatedAt    time.Time
	HTMLURL      string
	SnoozedUntil time.Time // zero if not currently snoozed
}

func (i Incident) IsSnoozed() bool {
	return !i.SnoozedUntil.IsZero() && i.SnoozedUntil.After(time.Now())
}

type OnCallEntry struct {
	ScheduleName string
	UserName     string
	EndTime      time.Time
}

type ScheduleEntry struct {
	UserID   string
	UserName string
	Start    time.Time
	End      time.Time
}

type ScheduleOverride struct {
	ID       string
	UserID   string
	UserName string
	Start    time.Time
	End      time.Time
}

type ScheduleAuditRecord struct {
	Action        string
	ActorName     string
	ExecutionTime time.Time
	ScheduleID    string
	ScheduleName  string
}
