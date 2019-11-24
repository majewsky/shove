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

//Event is a very minimal interface that mostly helps avoid using interface{}
//when passing around events of arbitrary types.
type Event interface {
	//Returns the event type that was passed to the EventDecoder which
	//instantiated this event.
	EventType() string
}

//EventDecoder is a type of function used by type Handler to decode events of
//different types. The payload argument contains the JSON body of the event's
//HTTP request. The eventType argument is the event type as specified by
//GitHub/Gitea, e.g. "push" or "fork". The possible Go types of events returned
//by the decoder depend on the decoder.
//
//There is no default catch-all decoder that decodes all events. Payloads have
//a *huge* amount of fields, and it's probably best for readability if you
//create your custom event types that decode just the fields you're interested
//in. A typical EventDecoder implementation looks something like this:
//
//	type FooEvent struct {
//	  Text string `json:"text"`
//	}
//	type BarEvent struct {
//	  Counter int `json:"counter"`
//	}
//
//	func MyEventDecoder(eventType string, payload []byte) (Event, error) {
//	  switch eventType {
//	  case "foo":
//	    e := FooEvent{}
//	    err := json.Unmarshal(payload, &e)
//	    return e, err
//	  case "bar":
//	    e := BarEvent{}
//	    err := json.Unmarshal(payload, &e)
//	    return e, err
//	  }
//	  return shove.MinimalEventDecoder(eventType)
//	}
//
//All custom event decoders should recognize at least the "ping" event type
//which is used by GitHub/Gitea to check the event delivery path to your
//application. As shown above, you can achieve this by using
//MinimalEventDecoder as a base.
//
//For unrecognized event types, (nil, nil) should be returned, which will cause
//the handler to not call its callback and return a standardized HTTP error
//response. If an error is returned, it will be written into the HTTP response
//body, and an error code of 401 (Bad Request) will be generated.
type EventDecoder func(eventType string, payload []byte) (Event, error)

//MinimalEventDecoder returns the string "ping" if eventType is "ping", and nil
//otherwise. See documentation on type EventDecoder for details.
func MinimalEventDecoder(eventType string, payload []byte) (Event, error) {
	if eventType == "ping" {
		return MinimalPingEvent{}, nil
	}
	return nil, nil
}

//MinimalPingEvent is returned by MinimalEventDecoder and corresponds to
//"X-GitHub-Event: ping".
type MinimalPingEvent struct{}

//EventType implements the Event interface.
func (MinimalPingEvent) EventType() string { return "ping" }
