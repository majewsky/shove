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

package shove

import "encoding/json"

//EventDecoder is a type of function used by type Handler to decode events of
//different types. The payload argument contains the JSON body of the event's
//HTTP request. The eventType argument is the event type as specified by
//GitHub, e.g. "push" or "fork". The possible Go types of events returned by
//the decoder depend on the decoder.
//
//For most users, DefaultEventDecoder should be be the right choice. If
//DefaultEventDecoder does not yet support a particular event type you're
//interested in, you can wrap it to add support for it:
//
//	type ExampleEvent struct {
//	  Text string `json:"text"`
//	}
//
//	func MyEventDecoder(eventType string, payload []byte) (interface{}, error) {
//	  if eventType == "example" {
//	    e := ExampleEvent{}
//	    err := json.Unmarshal(payload, &e)
//	    return e, err
//	  }
//	  return shove.DefaultEventDecoder(eventType)
//	}
//
//All custom event decoders should recognize at least the "ping" event type
//which is used by GitHub to check the event delivery path to your application.
//If you don't want to use the event types recognized by DefaultEventDecoder,
//you can use MinimalEventDecoder as a base. MinimalEventDecoder only recognizes
//"ping" events.
//
//For unrecognized event types, (nil, nil) should be returned, which will cause
//the handler to not call its callback and return a standardized HTTP error
//response. If an error is returned, it will be written into the HTTP response
//body, and an error code of 401 (Bad Request) will be generated.
type EventDecoder func(eventType string, payload []byte) (event interface{}, err error)

//DefaultEventDecoder maps event type strings used by GitHub onto the event
//types provided by this library. See documentation on type EventDecoder for
//details.
func DefaultEventDecoder(eventType string, payload []byte) (interface{}, error) {
	switch eventType {
	case "ping":
		e := PingEvent{}
		err := json.Unmarshal(payload, &e)
		return e, err
	default:
		return nil, nil
	}
}

//MinimalEventDecoder returns PingEvent if eventType is "ping", and nil
//otherwise. See documentation on type EventDecoder for details.
func MinimalEventDecoder(eventType string, payload []byte) (interface{}, error) {
	if eventType == "ping" {
		e := PingEvent{}
		err := json.Unmarshal(payload, &e)
		return e, err
	}
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////
// event types

//PingEvent corresponds to "X-GitHub-Event: ping".
type PingEvent struct {
}
