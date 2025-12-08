package main

import (
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/require"
)

func TestShouldProcessCommentNotification(t *testing.T) {
	p := &Plugin{}
	wh := &webhook{
		JiraWebhook: &JiraWebhook{
			Issue:   jira.Issue{ID: "ISSUE-1"},
			Comment: jira.Comment{ID: "10001"},
		},
		eventTypes: NewStringSet(eventCreatedComment),
	}

	require.True(t, p.shouldProcessCommentNotification(wh))
	require.False(t, p.shouldProcessCommentNotification(wh))

	p.recentCommentCacheLock.Lock()
	for k := range p.recentCommentCache {
		p.recentCommentCache[k] = time.Now().Add(-time.Second)
	}
	p.recentCommentCacheLock.Unlock()

	require.True(t, p.shouldProcessCommentNotification(wh))
}
