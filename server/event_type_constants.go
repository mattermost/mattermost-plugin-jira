package main

import "math"

const (
	eventCreated = uint64(1 << iota)
	eventCreatedComment
	eventDeleted
	eventDeletedComment
	eventDeletedUnresolved
	eventUpdatedAssignee
	eventUpdatedAttachment
	eventUpdatedComment
	eventUpdatedDescription
	eventUpdatedLabels
	eventUpdatedPriority
	eventUpdatedRank
	eventUpdatedReopened
	eventUpdatedResolved
	eventUpdatedSprint
	eventUpdatedStatus
	eventUpdatedSummary
	eventUpdatedIssuetype
	eventUpdatedCustomField
)

const maskLegacy = eventCreated |
	eventUpdatedReopened |
	eventUpdatedResolved |
	eventDeletedUnresolved

const maskComments = eventCreatedComment |
	eventDeletedComment |
	eventUpdatedComment

const maskDefault = maskLegacy |
	eventUpdatedAssignee

const maskAll = math.MaxUint64

const maskUpdatedAll = eventUpdatedAssignee |
	eventUpdatedAttachment |
	eventUpdatedComment |
	eventUpdatedDescription |
	eventUpdatedLabels |
	eventUpdatedPriority |
	eventUpdatedRank |
	eventUpdatedReopened |
	eventUpdatedResolved |
	eventUpdatedSprint |
	eventUpdatedStatus |
	eventUpdatedSummary |
	eventUpdatedIssuetype |
	eventUpdatedCustomField

const (
	eventCreatedStr            = "event_created"
	eventCreatedCommentStr     = "event_created_comment"
	eventDeletedStr            = "event_deleted"
	eventDeletedCommentStr     = "event_deleted_comment"
	eventUpdatedAllStr         = "event_updated_all"
	eventUpdatedAssigneeStr    = "event_updated_assignee"
	eventUpdatedAttachmentStr  = "event_updated_attachment"
	eventUpdatedCommentStr     = "event_updated_comment"
	eventUpdatedDescriptionStr = "event_updated_description"
	eventUpdatedLabelsStr      = "event_updated_labels"
	eventUpdatedPriorityStr    = "event_updated_priority"
	eventUpdatedRankStr        = "event_updated_rank"
	eventUpdatedReopenedStr    = "event_updated_reopened"
	eventUpdatedResolvedStr    = "event_updated_resolved"
	eventUpdatedSprintStr      = "event_updated_sprint"
	eventUpdatedStatusStr      = "event_updated_status"
	eventUpdatedSummaryStr     = "event_updated_summary"
)

var UI_ENUM_TO_MASK = map[string]uint64{
	eventCreatedStr:            eventCreated,
	eventCreatedCommentStr:     eventCreatedComment,
	eventDeletedStr:            eventDeleted | eventDeletedUnresolved,
	eventDeletedCommentStr:     eventDeletedComment,
	eventUpdatedAllStr:         maskUpdatedAll,
	eventUpdatedAssigneeStr:    eventUpdatedAssignee,
	eventUpdatedAttachmentStr:  eventUpdatedAttachment,
	eventUpdatedCommentStr:     eventUpdatedComment,
	eventUpdatedDescriptionStr: eventUpdatedDescription,
	eventUpdatedLabelsStr:      eventUpdatedLabels,
	eventUpdatedPriorityStr:    eventUpdatedPriority,
	eventUpdatedRankStr:        eventUpdatedRank,
	eventUpdatedReopenedStr:    eventUpdatedReopened,
	eventUpdatedResolvedStr:    eventUpdatedResolved,
	eventUpdatedSprintStr:      eventUpdatedSprint,
	eventUpdatedStatusStr:      eventUpdatedStatus,
	eventUpdatedSummaryStr:     eventUpdatedSummary,
}
