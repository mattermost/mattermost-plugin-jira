// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

type WebhookUser struct {
	Self         string
	Name         string
	Key          string
	EmailAddress string
	AvatarURLs   map[string]string
	DisplayName  string
	Active       bool
	TimeZone     string
}

type Webhook struct {
	WebhookEvent string
	Issue        struct {
		Self   string
		Key    string
		Fields struct {
			Assignee    *WebhookUser
			Reporter    *WebhookUser
			Summary     string
			Description string
			Priority    *struct {
				Id      string
				Name    string
				IconURL string
			}
			IssueType struct {
				Name    string
				IconURL string
			}
			Resolution *struct {
				Id string
			}
			Status struct {
				Id string
			}
			Labels []string
		}
	}
	User    WebhookUser
	Comment struct {
		Body         string
		UpdateAuthor WebhookUser
	}
	ChangeLog struct {
		Items []struct {
			From       string
			FromString string
			To         string
			ToString   string
			Field      string
		}
	}
	IssueEventTypeName string `json:"issue_event_type_name"`
}
