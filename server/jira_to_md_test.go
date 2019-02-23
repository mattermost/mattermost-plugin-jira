// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"regexp"
	"testing"
)

const (
	jiraBoldText          = "Don't forget to make *some bold text* to draw attention\r\n"
	jiraSubscriptText     = "Hiding ~submarines in the text~ is strategic.\r\n"
	jiraStrikethroughText = "A full count is three balls and -two strikes-.\r\n"
	jiraListText          = "# The numbered list\r\n# With two things\r\n# Actually three things are here\r\n# but wait, if you buy now, get fourth absolutely free\r\n"
)

func TestJiraToMarkdown(t *testing.T) {

	originalContent := "h1. This is a Heading 1\r\nh2. This is a Heading 2\r\nh3. This is a Heading 3\r\nh4. This is a heading 4\r\n\r\nSome {color:#14892c}festive green{color} color test\r\n\r\nSome {color:#d04437}festive red{color} color test\r\n\r\nThen _italicized text_ is here\r\n\r\nGood to have some +underlined text+ as well\r\n\r\nDon't forget to make *some bold text* to draw attention\r\n\r\nHaving ^super text^ is super\r\n\r\nHiding ~submarines in the text~ is strategic.\r\n\r\nA full count is three balls and -two strikes-.\r\n\r\n{{var i = ”code value”;}}\r\n\r\nMulti-line Go code snippet\r\n\r\n{code:go}\r\n// code comment\r\nfunc someFunc() string {\r\n  someVar := \"some var value\"\r\n  return someVar\r\n}\r\n{code}\r\n\r\n||Heading 1||Heading 2||\r\n|Col A1|Col A2|\r\n|Col B1|Col B2|\r\n\r\n\r\n{panel:title=My title}\r\nSome text with a title\r\n{panel}\r\n\r\n\r\n{noformat}\r\nSome pre-formatted text \r\n{noformat}\r\n\r\na link to [mattermost|https://mattermost.org]\r\n\r\n[http://example.com]\r\n\r\nA bullet list\r\n* A list of things\r\n* A second item in list\r\n* The third item in the list \r\n* The list of things fourth thing\r\n\r\nA numbered list\r\n# The numbered list\r\n# With two things\r\n# Actually three things are here\r\n# but wait, if you buy now, get fourth absolutely free\r\n\r\n"

	replacedContent := jiraToMarkdown(originalContent)

	for i := range jiraReplacers {

		// t.Logf("Testing: %s %s", jiraReplacers[i].Type, jiraReplacers[i].RegExp)

		re, err := regexp.Compile(jiraReplacers[i].RegExp)
		if err != nil {
			t.Error(err)
		}

		switch jiraReplacers[i].Type {

		// Bold requires separate test text due to Italic md being the same as jira
		case "Bold":
			if !re.MatchString(jiraBoldText) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, "not found in test content.")
			}

			replText := jiraToMarkdown(jiraBoldText)
			if replText == jiraBoldText {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, " was not replaced.")
			}

			// Subscript requires separate test text due to strikethrough
		case "Subscript":
			if !re.MatchString(jiraSubscriptText) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, "not found in test content.")
			}

			replText := jiraToMarkdown(jiraSubscriptText)
			if replText == jiraSubscriptText {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, " was not replaced.")
			}

			// Strikethrough requires separate test text due to subscript
		case "Strikethrough":
			if !re.MatchString(jiraStrikethroughText) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, "not found in test content.")
			}

			replText := jiraToMarkdown(jiraStrikethroughText)
			if replText == jiraStrikethroughText {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, " was not replaced.")
			}

			// Lists requires separate test text due to bold
		case "Lists":
			if !re.MatchString(jiraListText) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, "not found in test content.")
			}

			replText := jiraToMarkdown(jiraListText)
			if replText == jiraListText {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, " was not replaced.")
			}

		default:
			// Default handling of any other types

			if !re.MatchString(originalContent) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, "not found in test content.")
			}

			if re.MatchString(replacedContent) {
				t.Error(jiraReplacers[i].Type, ": RegExp", jiraReplacers[i].RegExp, " was not replaced.")
			}

		}

	}
}
