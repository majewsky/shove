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

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type testEvent struct {
	HookID int `json:"hook_id"`
}

//EventType implements the Event interface.
func (testEvent) EventType() string {
	return "ping"
}

func TestHandler(t *testing.T) {
	type receivedEvent struct {
		GUID       string
		WasPointer bool
		Event      testEvent
	}

	testCases := []struct {
		Method       string
		Headers      map[string]string
		Body         string
		Expected     *receivedEvent
		ResponseCode int
		ResponseBody string
	}{
		//case 1a: success case with GitHub-style signature
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "first",
				"X-GitHub-Event":    "ping",
				"X-Hub-Signature":   "sha1=71652c35709ccaec5fb1de93c576d27ab4325273",
			},
			//NOTE: When changing the body, you need to recompute the signature
			//above, using the secret key "verysecret".
			Body: `{"hook_id":42}`,
			Expected: &receivedEvent{
				GUID:       "first",
				WasPointer: false,
				Event:      testEvent{HookID: 42},
			},
			ResponseCode: 204,
		},
		//case 1b: success case with Gitea-style signature (Gitea uses HMAC-SHA256
		//instead of HMAC-SHA1, but otherwise generates the X-GitHub-... headers
		//and request body in the same way as GitHub for compatibility purposes)
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "first",
				"X-GitHub-Event":    "ping",
				"X-Gitea-Signature": "63e8905ed20fa4cef5b1be5dc6a111615b04e1f7ffc43716ade5b7b27f93b17d",
			},
			Body: `{"hook_id":42}`,
			Expected: &receivedEvent{
				GUID:       "first",
				WasPointer: false,
				Event:      testEvent{HookID: 42},
			},
			ResponseCode: 204,
		},
		//case 2a/b: like case 1a/b, but broken HMAC
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "second",
				"X-GitHub-Event":    "ping",
				"X-Hub-Signature":   "sha1=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			Body:         `{"hook_id":42}`,
			ResponseCode: 401,
			ResponseBody: "invalid signature header",
		},
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "second",
				"X-GitHub-Event":    "ping",
				"X-Gitea-Signature": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			Body:         `{"hook_id":42}`,
			ResponseCode: 401,
			ResponseBody: "invalid signature header",
		},
		//case 3a/b: like case 1a/b, but malformed HMAC (does not even look like a valid HMAC-SHA1 or HMAC-SHA256)
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "third",
				"X-GitHub-Event":    "ping",
				"X-Hub-Signature":   "sha1=42",
			},
			Body:         `{"hook_id":42}`,
			ResponseCode: 401,
			ResponseBody: "invalid signature header",
		},
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "third",
				"X-GitHub-Event":    "ping",
				"X-Gitea-Signature": "42",
			},
			Body:         `{"hook_id":42}`,
			ResponseCode: 401,
			ResponseBody: "invalid signature header",
		},
		//case 4: error during json.Unmarshal
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "fourth",
				"X-GitHub-Event":    "ping",
				"X-Hub-Signature":   "sha1=6cd3585b909ed39ca1107f890b3438e0d0b5b04d",
			},
			//NOTE: When changing the body, you need to recompute the signature
			//above, using the secret key "verysecret".
			Body:         `{"hook_id":"foobar"}`,
			ResponseCode: 400,
			ResponseBody: "json: cannot unmarshal string into Go struct field testEvent.hook_id of type int",
		},
		//case 5: like case 1a, but invalid method
		{
			Method: "GET",
			Headers: map[string]string{
				"X-GitHub-Delivery": "fifth",
				"X-GitHub-Event":    "ping",
				"X-Hub-Signature":   "sha1=71652c35709ccaec5fb1de93c576d27ab4325273",
			},
			//NOTE: When changing the body, you need to recompute the signature
			//above, using the secret key "verysecret".
			Body:         `{"hook_id":42}`,
			ResponseCode: 405,
			ResponseBody: "method not allowed",
		},
		//case 6: like case 1a, but unknown X-GitHub-Event type
		{
			Method: "POST",
			Headers: map[string]string{
				"X-GitHub-Delivery": "sixth",
				"X-GitHub-Event":    "test",
				"X-Hub-Signature":   "sha1=71652c35709ccaec5fb1de93c576d27ab4325273",
			},
			//NOTE: When changing the body, you need to recompute the signature
			//above, using the secret key "verysecret".
			Body:         `{"hook_id":42}`,
			ResponseCode: 501,
			ResponseBody: "event type not supported",
		},
	}

	var receivedEvents []receivedEvent
	handler := Handler{
		SecretKey: "verysecret",
		EventDecoder: func(eventType string, payload []byte) (Event, error) {
			if eventType == "ping" {
				e := testEvent{}
				err := json.Unmarshal(payload, &e)
				return e, err
			}
			return nil, nil
		},
		Callback: func(guid string, event Event) {
			switch event := event.(type) {
			case testEvent:
				receivedEvents = append(receivedEvents, receivedEvent{guid, false, event})
			case *testEvent:
				receivedEvents = append(receivedEvents, receivedEvent{guid, true, *event})
			default:
				t.Errorf("unexpected event type: %T", event)
			}
		},
	}

	for idx, tc := range testCases {
		//reset test harness
		receivedEvents = nil
		if t.Failed() {
			t.FailNow()
		}

		//execute request
		req := httptest.NewRequest(tc.Method, "/", strings.NewReader(tc.Body))
		for k, v := range tc.Headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		//check for correct HTTP response
		if rec.Code != tc.ResponseCode {
			t.Errorf("test case %d: expected response code %d, got %d", idx, tc.ResponseCode, rec.Code)
		}
		responseBody := strings.TrimSpace(rec.Body.String())
		if responseBody != tc.ResponseBody {
			t.Errorf("test case %d: expected response body %q, got %q", idx, tc.ResponseBody, responseBody)
		}

		//check for correct event being generated
		if tc.Expected == nil {
			if len(receivedEvents) > 0 {
				t.Errorf("test case %d: expected no events, but got %#v", idx, receivedEvents)
			}
		} else {
			switch len(receivedEvents) {
			case 0:
				t.Errorf("test case %d: expected event %#v, but got none", idx, *tc.Expected)
			case 1:
				if !reflect.DeepEqual(receivedEvents[0], *tc.Expected) {
					t.Errorf("test case %d: expected event %#v, but got %#v", idx, *tc.Expected, receivedEvents[0])
				}
			default:
				t.Errorf("test case %d: expected one event, but got multiple: %#v", idx, receivedEvents)
			}
		}
	}
}
