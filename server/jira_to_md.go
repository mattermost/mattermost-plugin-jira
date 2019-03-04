// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

/*
JIRA Text Formatting Reference
https://jira.atlassian.com/secure/WikiRendererHelpAction.jspa?section=all
*/

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type jiraReplacer struct {
	Type           string
	RegExp         *regexp.Regexp
	Replace        string
	ReplaceStrFunc func(string) string // Takes priority
}

var (

	// Headers ReplaceFunc Regular Expression
	headersReplaceFuncRegExp = regexp.MustCompile(`h([1-6]+)\. (.*?)(\r|\n)`)

	// Table Headers ReplaceFunc Regular Expression
	tableHeadersReplaceFuncRegExp = regexp.MustCompile(`\|([^|]+)`)

	jiraReplacers = []jiraReplacer{

		{
			Type:           "Lists",
			RegExp:         regexp.MustCompile(`(m?)[ ^\t]*(#+)\s+`),
			ReplaceStrFunc: replaceNumberedListItems,
		},

		// Headers
		{
			Type:           "Headers",
			RegExp:         regexp.MustCompile(`h([1-6]+)\. (.*?)(\r|\n)`),
			ReplaceStrFunc: replaceHeaders,
		},

		// Bold
		{
			Type:    "Bold",
			RegExp:  regexp.MustCompile(`\*(\S.*?)\*`),
			Replace: "**${1}**",
		},

		// Italic (same in jira and md)
		{
			Type:    "Italic",
			RegExp:  regexp.MustCompile(`\_(\S.*?)\_`),
			Replace: "*${1}*",
		},

		// Monospaced text
		{
			Type:    "Monospaced",
			RegExp:  regexp.MustCompile(`\{\{([^}]+)\}\}`),
			Replace: "`${1}`",
		},

		// // Underline (not a thing in md)
		// jiraReplacer{
		//  RegExp: regexp.MustCompile( `\+([^+]*)\+`),
		//  Replace: "${1}",
		// },

		// Citations (buggy)
		// \?\?((?:.[^?]|[^?].)+)\?\?
		// '<cite>$1</cite>'

		// Superscript
		{
			Type:    "Superscript",
			RegExp:  regexp.MustCompile(`\^([^^]*)\^`),
			Replace: "<sup>${1}</sup>",
		},

		// Subscript
		{
			Type:    "Subscript",
			RegExp:  regexp.MustCompile(`~([^~]*)~`),
			Replace: "<sub>${1}</sub>",
		},

		// Strikethrough
		{
			Type:    "Strikethrough",
			RegExp:  regexp.MustCompile(`\s-(\S+.*?\S)-(?s)`),
			Replace: " ~~${1}~~ ",
		},

		// Code Block
		{
			Type: "Code Block",
			// RegExp: regexp.MustCompile(`(?s){code(:([a-z]+))?}(.*){code}`),
			RegExp:  regexp.MustCompile(`(?s){code(:([a-z]+))?([:|]?(title|borderStyle|borderColor|borderWidth|bgColor|titleBGColor)=.+?)*}(.*?){code}`),
			Replace: "```${2}${5}```",
		},

		// Pre-formatted text
		{
			Type:    "Pre-formatted",
			RegExp:  regexp.MustCompile(`{noformat}`),
			Replace: "```",
		},

		// Named Links
		{
			Type:    "Named Link",
			RegExp:  regexp.MustCompile(`\[(.+?)\|(.*?)\]`),
			Replace: "[${1}](${2})",
		},

		// Un-named Links
		// jiraReplacer{
		// 	Type:    "Un-named Link",
		// 	RegExp: regexp.MustCompile( `\[([^|]+)\]`),
		// 	Replace: "<${1}>${2}",
		// },

		// // Single Paragraph Blockquote
		// jiraReplacer{
		//  RegExp: regexp.MustCompile( `^bq\.\s+`),
		//  Replace: "> ",
		// },

		// Remove color: unsupported in md
		{
			Type:    "Text Color",
			RegExp:  regexp.MustCompile(`{color:.+}(.*){color}`),
			Replace: "${1}",
		},

		// // panel into table
		{
			Type:    "Panel (to table)",
			RegExp:  regexp.MustCompile(`(?s){panel:title=([^}]*)}\n?(.*?)\n?{panel}`),
			Replace: "\n| ${1} |\n| --- |\n| ${2} |",
		},

		// table header
		{
			Type:           "Table Header",
			RegExp:         regexp.MustCompile(`[ \t]*((?:\|\|.*?)+\|\|)[ \t]*`),
			ReplaceStrFunc: replaceTableHeaders,
		},

		// remove leading-space of table headers and rows
		// "|": `^[ \t]*\|`,

	}
)

func jiraToMarkdown(body string) string {

	result := body

	for i := range jiraReplacers {

		if jiraReplacers[i].RegExp == nil {
			continue
		}

		if jiraReplacers[i].ReplaceStrFunc != nil {
			result = jiraReplacers[i].RegExp.ReplaceAllStringFunc(result, jiraReplacers[i].ReplaceStrFunc)
			continue
		}

		if jiraReplacers[i].Replace != "" {
			result = jiraReplacers[i].RegExp.ReplaceAllString(result, jiraReplacers[i].Replace)
			continue
		}

	}

	return result
}

func replaceTableHeaders(repl string) string {

	repl = strings.Replace(repl, "||", "|", -1)

	headers := tableHeadersReplaceFuncRegExp.ReplaceAllString(repl, "| ${1} ")

	repl = fmt.Sprintf("\n%s\n%s", headers, tableHeadersReplaceFuncRegExp.ReplaceAllString(repl, "| --- "))

	return repl
}

func replaceNumberedListItems(repl string) string {

	repl = strings.TrimLeft(strings.TrimSpace(repl), "#")

	return fmt.Sprintf("1. %s", repl)

}

func replaceHeaders(repl string) string {

	level, _ := strconv.Atoi(headersReplaceFuncRegExp.ReplaceAllString(repl, "${1}"))
	content := headersReplaceFuncRegExp.ReplaceAllString(repl, "${2}\n")

	if level == 0 {
		return content
	}

	str := "#"
	for level > 1 {
		str += "#"
		level--
	}

	return fmt.Sprintf("%s %s", str, content)
}
