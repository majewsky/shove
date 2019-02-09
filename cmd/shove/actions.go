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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/majewsky/shove"
	"github.com/sapcc/go-bits/logg"
)

////////////////////////////////////////////////////////////////////////////////
// Action

//Action is an action that can be taken upon receiving a matching event.
type Action struct {
	Name     string `yaml:"name"`
	Triggers []struct {
		EventTypes    []string `yaml:"events"`
		FullRepoNames []string `yaml:"repos"`
	} `yaml:"on"`
	RunTask struct {
		Command []string `yaml:"command"`
	} `yaml:"run"`
}

//Matches checks if the given event matches one of the triggers of this action.
func (a Action) Matches(event Event) bool {
	for _, t := range a.Triggers {
		if containsString(t.EventTypes, event.EventType()) {
			//for pseudo-events, FullRepoNames must be empty
			fullRepoName := event.FullRepoName()
			if fullRepoName == "" {
				return true
			}
			//for regular events, trigger must match the repo name
			if containsString(t.FullRepoNames, fullRepoName) {
				return true
			}
		}
	}
	return false
}

func containsString(list []string, val string) bool {
	for _, item := range list {
		if item == val {
			return true
		}
	}
	return false
}

//Execute runs the tasks in this action.
func (a Action) Execute(guid string, event Event) {
	logg.Info("[%s] executing action: %s", guid, a.Name)

	//This is written such that other types of tasks can be added later.
	if len(a.RunTask.Command) > 0 {
		cmd := exec.Command(a.RunTask.Command[0], a.RunTask.Command[1:]...)
		cmd.Stdin = nil
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Env = os.Environ()
		for k, v := range event.EnvVariables() {
			cmd.Env = append(cmd.Env, k+"="+v)
		}

		err := cmd.Run()
		if err != nil {
			logg.Error("[%s] command %v failed: %s", guid, a.RunTask.Command, err.Error())
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Configuration

//Configuration contains the contents of the $SHOVE_CONFIG file.
type Configuration struct {
	Actions []Action `yaml:"actions"`
}

//Validate checks the configuration for semantic errors that the YAML decoder cannot detect.
func (c Configuration) Validate() (errs []error) {
	for aIdx, action := range c.Actions {
		if action.Name == "" {
			errs = append(errs, fmt.Errorf("actions[%d].name may not be empty", aIdx))
		}
		if len(action.Triggers) == 0 {
			errs = append(errs, fmt.Errorf("actions[%d].on may not be empty", aIdx))
		}

		for tIdx, trigger := range action.Triggers {
			if len(trigger.EventTypes) == 0 {
				errs = append(errs, fmt.Errorf("actions[%d].on[%d].events may not be empty", aIdx, tIdx))
			}

			var pseudoEvents []string
			for _, eventType := range trigger.EventTypes {
				if !isSupportedEventType(eventType) {
					errs = append(errs, fmt.Errorf("actions[%d].on[%d].events contains unsupported event type %q", aIdx, tIdx, eventType))
				}
				if strings.HasPrefix(eventType, "shove-") {
					pseudoEvents = append(pseudoEvents, eventType)
				}
			}

			if len(pseudoEvents) > 0 && len(trigger.FullRepoNames) > 0 {
				errs = append(errs, fmt.Errorf("actions[%d].on[%d] matches pseudo-events %v, but also requires a match on repository names", aIdx, tIdx, pseudoEvents))
			}
		}

		if len(action.RunTask.Command) == 0 {
			errs = append(errs, fmt.Errorf("actions[%d].execute is missing", aIdx))
		}
	}
	return
}

//HandleEvent satisfies the shove.Handler.Callback contract.
func (c Configuration) HandleEvent(guid string, e shove.Event) {
	//skip ping events
	event, ok := e.(Event)
	if !ok {
		return
	}

	//report event in log
	fullRepoName := event.FullRepoName()
	if fullRepoName == "" {
		logg.Info("[%s] received %s event", guid, e.EventType())
	} else {
		logg.Info("[%s] received %s event for %s", guid, e.EventType(), fullRepoName)
	}

	for _, action := range c.Actions {
		if action.Matches(event) {
			action.Execute(guid, event)
		}
	}
}
