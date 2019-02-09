/******************************************************************************
*
*  Copyright 2019 Stefan Majewsky <majewsky@gmx.net>
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
******************************************************************************/

package main

import (
	"encoding/json"
	"strings"

	"github.com/majewsky/shove"
)

//Event extends shove.Event.
type Event interface {
	shove.Event
	FullRepoName() string
	EnvVariables() map[string]string
}

var supportedEventTypes = []Event{
	PushEvent{},
	ShoveStartupEvent{},
}

func isSupportedEventType(eventType string) bool {
	for _, e := range supportedEventTypes {
		if e.EventType() == eventType {
			return true
		}
	}
	return false
}

func decodeEvent(eventType string, payload []byte) (shove.Event, error) {
	switch eventType {
	case "push":
		e := PushEvent{}
		err := json.Unmarshal(payload, &e)
		if err == nil {
			e.RawMessage = payload
			if strings.HasPrefix(e.Ref, "refs/heads/") {
				e.Branch = strings.TrimPrefix(e.Ref, "refs/heads/")
			}
		}
		return e, err
	default:
		return shove.MinimalEventDecoder(eventType, payload)
	}
}

////////////////////////////////////////////////////////////////////////////////

//PushEvent corresponds to "X-GitHub-Event: push".
type PushEvent struct {
	Ref        string `json:"ref"`
	Commit     string `json:"after"`
	Branch     string `json:"-"` //If .Ref looks like "refs/heads/foo/bar", .Branch contains only the branch name (in this example, "foo/bar"). Otherwise, .Branch is empty.
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"repository"`
	RawMessage []byte `json:"-"`
}

//EventType implements the shove.Event interface.
func (PushEvent) EventType() string {
	return "push"
}

//FullRepoName implements the Event interface.
func (e PushEvent) FullRepoName() string {
	return e.Repository.Owner.Name + "/" + e.Repository.Name
}

//EnvVariables implements the Event interface.
func (e PushEvent) EnvVariables() map[string]string {
	return map[string]string{
		"SHOVE_VAR_REF":        e.Ref,
		"SHOVE_VAR_BRANCH":     e.Branch,
		"SHOVE_VAR_COMMIT":     e.Commit,
		"SHOVE_VAR_REPO_NAME":  e.Repository.Name,
		"SHOVE_VAR_REPO_OWNER": e.Repository.Owner.Name,
		"SHOVE_PAYLOAD":        string(e.RawMessage),
	}
}

////////////////////////////////////////////////////////////////////////////////

//ShoveStartupEvent is a pseudo-event that fires once on startup.
type ShoveStartupEvent struct{}

//EventType implements the Event interface.
func (ShoveStartupEvent) EventType() string {
	return "shove-startup"
}

//FullRepoName implements the Event interface.
func (ShoveStartupEvent) FullRepoName() string {
	return ""
}

//EnvVariables implements the Event interface.
func (ShoveStartupEvent) EnvVariables() map[string]string {
	return nil
}
