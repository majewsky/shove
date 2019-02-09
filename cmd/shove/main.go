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
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/majewsky/shove"
	"github.com/sapcc/go-bits/logg"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	//read SHOVE_CONFIG
	configPath := os.Getenv("SHOVE_CONFIG")
	if configPath == "" {
		configPath = "./shove.yaml"
	}
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.Getenv("SHOVE_CONFIG") == "" {
			logg.Info("defaulting to SHOVE_CONFIG=./shove.yaml")
		}
		logg.Fatal(err.Error())
	}
	var config Configuration
	err = yaml.UnmarshalStrict(configBytes, &config)
	if err != nil {
		logg.Fatal(err.Error())
	}
	errs := config.Validate()
	if len(errs) > 0 {
		for _, err = range errs {
			logg.Error(err.Error())
		}
		os.Exit(1)
	}

	h := shove.Handler{
		EventDecoder: decodeEvent,
		Callback:     config.HandleEvent,
	}

	//read SHOVE_SECRET
	h.SecretKey = os.Getenv("SHOVE_SECRET")
	if h.SecretKey == "" {
		logg.Fatal("missing environment variable: SHOVE_SECRET")
	}

	//read SHOVE_PORT
	portStr := os.Getenv("SHOVE_PORT")
	if portStr == "" {
		logg.Fatal("missing environment variable: SHOVE_PORT")
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		logg.Fatal("invalid SHOVE_PORT: %s\n", err.Error())
	}

	//ensure that child processes do not see our secrets
	os.Unsetenv("SHOVE_CONFIG")
	os.Unsetenv("SHOVE_SECRET")
	os.Unsetenv("SHOVE_PORT")

	//emit the shove-startup event
	config.HandleEvent("00000000-0000-0000-0000-000000000000", ShoveStartupEvent{})

	//listen for events
	http.Handle("/", h)
	logg.Fatal("%v", http.ListenAndServe(":"+strconv.FormatUint(port, 10), nil))
}
