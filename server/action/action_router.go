// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"strings"
)

type ActionRouter struct {
	findHandler    func(route string) ActionFunc
	defaultHandler ActionFunc
	logHandler     ActionFunc
}

func (ar ActionRouter) RunRoute(route string, a Action, ac *ActionContext) {
	route = strings.TrimRight(route, "/")
	// See if we have a handler for the exact route match
	handler := ar.findHandler(route)
	if handler == nil {
		// Look for a subpath match
		handler = ar.findHandler(route + "/*")
	}
	// Look for a /* above
	for handler == nil {
		n := strings.LastIndex(route, "/")
		if n == -1 {
			break
		}
		handler = ar.findHandler(route[:n] + "/*")
		route = route[:n]
	}
	// Use the default, if needed
	if handler == nil {
		handler = ActionScript{ar.DefaultRouteHandler}
	}

	// Run the handler
	err := handler.Run(a, ac)
	if err != nil {
		return
	}

	// Log
	if ar.LogFilter != nil {
		_ = ar.LogFilter(a, ac)
	}
}

type ActionScript []ActionFunc

func (script ActionScript) Run(a Action, ac *ActionContext) error {
	for _, f := range script {
		if f == nil {
			continue
		}
		err := f(a, ac)
		if err != nil {
			ac.LogErr = err
			return err
		}
	}
	return nil
}
