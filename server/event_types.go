package main

type EventTypeSet map[string]bool

func (set EventTypeSet) Copy() EventTypeSet {
	result := EventTypeSet{}
	for k, v := range set {
		result[k] = v
	}

	return result
}

func (set EventTypeSet) Contains(eventType string) bool {
	_, exists := set[eventType]

	return exists
}

func (set EventTypeSet) Add(eventType string) {
	set[eventType] = true
}

func (set EventTypeSet) Union(set2 EventTypeSet) EventTypeSet {
	result := EventTypeSet{}

	for k, _ := range set {
		result[k] = true
	}

	for k, _ := range set2 {
		result[k] = true
	}

	return result
}

func (set EventTypeSet) Intersection(set2 EventTypeSet) EventTypeSet {
	result := EventTypeSet{}
	for k, _ := range set {
		if set2.Contains(k) {
			result.Add(k)
		}
	}

	return result
}

func (set EventTypeSet) HasIntersection(set2 EventTypeSet) bool {
	for k, _ := range set {
		if set2.Contains(k) {
			return true
		}
	}

	return false
}

const (
	eventCreated            = "event_created"
	eventCreatedComment     = "event_created_comment"
	eventDeleted            = "event_deleted"
	eventDeletedUnresolved  = "event_deleted_unresolved"
	eventDeletedComment     = "event_deleted_comment"
	eventUpdatedAll         = "event_updated_all"
	eventUpdatedAssignee    = "event_updated_assignee"
	eventUpdatedAttachment  = "event_updated_attachment"
	eventUpdatedComment     = "event_updated_comment"
	eventUpdatedDescription = "event_updated_description"
	eventUpdatedLabels      = "event_updated_labels"
	eventUpdatedPriority    = "event_updated_priority"
	eventUpdatedRank        = "event_updated_rank"
	eventUpdatedReopened    = "event_updated_reopened"
	eventUpdatedResolved    = "event_updated_resolved"
	eventUpdatedSprint      = "event_updated_sprint"
	eventUpdatedStatus      = "event_updated_status"
	eventUpdatedSummary     = "event_updated_summary"
	eventUpdatedIssuetype   = "event_updated_issue_type"
	eventUpdatedFixVersion  = "event_updated_fix_version"
)

var maskLegacy = EventTypeSet{
	eventCreated:           true,
	eventUpdatedReopened:   true,
	eventUpdatedResolved:   true,
	eventDeletedUnresolved: true,
}

var maskComments = EventTypeSet{
	eventCreatedComment: true,
	eventDeletedComment: true,
	eventUpdatedComment: true,
}

var maskDefault = EventTypeSet{
	eventUpdatedAssignee: true,
}.Union(maskLegacy)

var maskAll = EventTypeSet{
	eventCreated:            true,
	eventCreatedComment:     true,
	eventDeleted:            true,
	eventDeletedUnresolved:  true,
	eventDeletedComment:     true,
	eventUpdatedAll:         true,
	eventUpdatedAssignee:    true,
	eventUpdatedAttachment:  true,
	eventUpdatedComment:     true,
	eventUpdatedDescription: true,
	eventUpdatedLabels:      true,
	eventUpdatedPriority:    true,
	eventUpdatedRank:        true,
	eventUpdatedReopened:    true,
	eventUpdatedResolved:    true,
	eventUpdatedSprint:      true,
	eventUpdatedStatus:      true,
	eventUpdatedSummary:     true,
	eventUpdatedIssuetype:   true,
	eventUpdatedFixVersion:  true,
}
