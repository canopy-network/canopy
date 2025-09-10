package lib

type EventType string

const (
	EventTypeReward             EventType = "reward"
	EventTypeSlash              EventType = "slash"
	EventTypeAutomaticPause     EventType = "automatic-pause"
	EventTypeAutomaticUnstaking EventType = "automatic-unstaking"
	EventTypeAutomaticUnstake   EventType = "automatic-unstake"
)

type Events []*Event

func (e *Events) Len() int      { return len(*e) }
func (e *Events) New() Pageable { return &Events{} }
