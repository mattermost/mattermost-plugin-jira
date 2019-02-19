// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type jiraReplacer struct {
	Type           string
	RegExp         string
	Replace        string
	ReplaceStrFunc func(string) string // Takes priority
}

var (
	jiraReplacers = []jiraReplacer{

		// Headers
		jiraReplacer{
			Type:           "Headers",
			RegExp:         `h([0-6]+)\. (.*?)(\r|\n)`,
			ReplaceStrFunc: replaceHeaders,
		},

		// Bold
		jiraReplacer{
			Type:    "Bold",
			RegExp:  `\*(\S.*?)\*`,
			Replace: "**${1}**",
		},

		// Italic (same in jira and md)
		jiraReplacer{
			Type:    "Italic",
			RegExp:  `\_(\S.*?)\_`,
			Replace: "*${1}*",
		},

		// Monospaced text
		jiraReplacer{
			Type:    "Monospaced",
			RegExp:  `\{\{([^}]+)\}\}`,
			Replace: "`${1}`",
		},

		// // Underline
		// jiraReplacer{
		//  RegExp:  `\+([^+]*)\+`,
		//  Replace: "${1}",
		// },

		// Citations (buggy)
		//.replace(/\?\?((?:.[^?]|[^?].)+)\?\?/g, '<cite>$1</cite>')

		// Superscript
		jiraReplacer{
			Type:    "Superscript",
			RegExp:  `\^([^^]*)\^`,
			Replace: "<sup>${1}</sup>",
		},

		// Subscript
		jiraReplacer{
			Type:    "Subscript",
			RegExp:  `~([^~]*)~`,
			Replace: "<sub>${1}</sub>",
		},

		// Strikethrough
		jiraReplacer{
			Type:    "Strikethrough",
			RegExp:  `\s-(\S+.*?\S)-(?s)`,
			Replace: " ~~${1}~~ ",
		},

		// Code Block
		jiraReplacer{
			Type: "Code Block",
			// RegExp: `(?s){code(:([a-z]+))?}(.*){code}`,
			RegExp:  `(?s){code(:([a-z]+))?([:|]?(title|borderStyle|borderColor|borderWidth|bgColor|titleBGColor)=.+?)*}(.*?){code}`,
			Replace: "```${2}${5}```",
		},

		// Pre-formatted text
		jiraReplacer{
			Type:    "Pre-formatted",
			RegExp:  `{noformat}`,
			Replace: "```",
		},

		// Named Links
		jiraReplacer{
			Type:    "Named Link",
			RegExp:  `\[(.+?)\|(.*?)\]`,
			Replace: "[${1}](${2})",
		},

		// Un-named Links
		// jiraReplacer{
		// 	Type:    "Un-named Link",
		// 	RegExp:  `\[([^|]+)\]`,
		// 	Replace: "<${1}>${2}",
		// },

		// // Single Paragraph Blockquote
		// jiraReplacer{
		//  RegExp:  `^bq\.\s+`,
		//  Replace: "> ",
		// },

		// Remove color: unsupported in md
		jiraReplacer{
			Type:    "Text Color",
			RegExp:  `{color:.+}(.*){color}`,
			Replace: "${1}",
		},

		// // panel into table
		jiraReplacer{
			Type:    "Panel (to table)",
			RegExp:  `(?s){panel:title=([^}]*)}\n?(.*?)\n?{panel}`,
			Replace: "\n| ${1} |\n| --- |\n| ${2} |",
		},

		// table header
		jiraReplacer{
			Type:           "Table Header",
			RegExp:         `[ \t]*((?:\|\|.*?)+\|\|)[ \t]*`,
			ReplaceStrFunc: replaceTableHeaders,
		},

		// remove leading-space of table headers and rows
		// "|": `^[ \t]*\|`,
	}
)

func jiraToMarkdown(body string) string {

	result := body

	for i := range jiraReplacers {

		re, err := regexp.Compile(jiraReplacers[i].RegExp)
		if err != nil {
			fmt.Println(jiraReplacers[i].Type, "RegExp Error:", err.Error())
			continue
		}

		if jiraReplacers[i].ReplaceStrFunc != nil {

			result = re.ReplaceAllStringFunc(result, jiraReplacers[i].ReplaceStrFunc)
			continue

		}

		if jiraReplacers[i].Replace != "" {

			result = re.ReplaceAllString(result, jiraReplacers[i].Replace)
			continue
		}

	}

	return result
}

func replaceTableHeaders(repl string) string {

	repl = strings.Replace(repl, "||", "|", -1)

	re := regexp.MustCompile(`\|([^|]+)`)

	headers := re.ReplaceAllString(repl, "| ${1} ")

	repl = fmt.Sprintf("\n%s\n%s", headers, re.ReplaceAllString(repl, "| --- "))

	return repl
}

func replaceHeaders(repl string) string {
	re := regexp.MustCompile(`h([0-6]+)\. (.*?)(\r|\n)`)

	levelStr := re.ReplaceAllString(repl, "${1}")
	level, _ := strconv.Atoi(levelStr)
	content := re.ReplaceAllString(repl, "${2}")

	return fmt.Sprintf("%s %s", strings.Join(make([]string, level+1), "#"), content)
}
