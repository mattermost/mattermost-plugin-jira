// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// HTTPAction and CommandAction are declared public so that the plugin can access their
// internals in special cases, Action interface is not mature enough.
type CommandAction struct {
	CommandMetadata
	action

	Args            []string
	CommandArgs     *model.CommandArgs
	CommandResponse *model.CommandResponse
}

type CommandMetadata struct {
	// MinTotalArgs and MaxTotalArgs are applied to the total number of
	// whitespace-separated tokens, including the `/jira` and everything after
	// it.
	MinTotalArgs int
	MaxTotalArgs int

	// ArgNames are for the acual arguments of the command, in the order in
	// which they must appear.
	ArgNames []string
}

var _ Action = (*CommandAction)(nil)

func NewCommandAction(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs,
	commandMetadata CommandMetadata) (*CommandAction, *ActionContext) {
	action := *newAction(p, c, commandArgs.UserId)
	return &CommandAction{
		action:          action,
		CommandArgs:     commandArgs,
		CommandMetadata: commandMetadata,
	}, &action.ActionContext
}

func ParseCommmandAction(

	_ key) string {
	if key == "" || key[0] != '$' {
		return ""
	}
	n, _ := strconv.Atoi(key[1:])
	if n < 1 || n > len(commandAction.Args) {
		return ""
	}
	return commandAction.Args[n-1]
}

func (commandAction CommandAction) RespondError(code int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err != nil {
		commandAction.CommandResponse = commandResponsef(err.Error())
	}
	return err
}

func (commandAction CommandAction) RespondPrintf(format string, args ...interface{}) error {
	commandAction.CommandResponse = commandResponsef(format, args...)
	return nil
}

func (commandAction CommandAction) RespondRedirect(redirectURL string) error {
	commandAction.CommandResponse = &model.CommandResponse{
		GotoLocation: redirectURL,
	}
	return nil
}

func (commandAction CommandAction) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := commandAction.PluginConfig.Templates[templateKey]
	if t == nil {
		return commandAction.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	bb := &bytes.Buffer{}
	err := t.Execute(bb, values)
	if err != nil {
		return commandAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	commandAction.CommandResponse = commandResponsef(string(bb.Bytes()))
	return nil
}

func (commandAction CommandAction) RespondJSON(value interface{}) error {
	bb, err := json.Marshal(value)
	if err != nil {
		return commandAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	commandAction.CommandResponse = commandResponsef(string(bb))
	return nil
}
