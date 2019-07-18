package main

const (
	eventCreated            = "event_created"
	eventCreatedComment     = "event_created_comment"
	eventDeleted            = "event_deleted"
	eventDeletedUnresolved  = "event_deleted_unresolved"
	eventDeletedComment     = "event_deleted_comment"
	eventUpdatedAny         = "event_updated_any"
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
	eventUpdatedReporter    = "event_updated_fix_versions"
)

var legacyEvents = NewStringSet(
	eventCreated,
	eventUpdatedReopened,
	eventUpdatedResolved,
	eventDeletedUnresolved,
)

var commentEvents = NewStringSet(
	eventCreatedComment,
	eventDeletedComment,
	eventUpdatedComment,
)

var defaultEvents = legacyEvents.Add(eventUpdatedAssignee)

var allEvents = NewStringSet(
	eventCreated,
	eventCreatedComment,
	eventDeleted,
	eventDeletedUnresolved,
	eventDeletedComment,
	eventUpdatedAny,
	eventUpdatedAssignee,
	eventUpdatedAttachment,
	eventUpdatedComment,
	eventUpdatedDescription,
	eventUpdatedLabels,
	eventUpdatedPriority,
	eventUpdatedRank,
	eventUpdatedReopened,
	eventUpdatedResolved,
	eventUpdatedSprint,
	eventUpdatedStatus,
	eventUpdatedSummary,
	eventUpdatedIssuetype,
	eventUpdatedFixVersion,
)

var updateEvents = NewStringSet(
	eventUpdatedAssignee,
	eventUpdatedAttachment,
	eventUpdatedComment,
	eventUpdatedDescription,
	eventUpdatedLabels,
	eventUpdatedPriority,
	eventUpdatedRank,
	eventUpdatedReopened,
	eventUpdatedResolved,
	eventUpdatedSprint,
	eventUpdatedStatus,
	eventUpdatedSummary,
	eventUpdatedIssuetype,
	eventUpdatedFixVersion,
)
