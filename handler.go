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

//Package shove is a library for receiving GitHub webhooks. If you don't know
//what webhooks are, check out GitHub's documentation first:
//<https://developer.github.com/webhooks/>
package shove

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

//Handler is an http.Handler that receives GitHub webhooks. It does not match
//on paths, so you might want to wrap it in a router that does.
type Handler struct {
	//The secret key that GitHub uses to sign events for this webhook.
	SecretKey string
	//A mapper function that maps GitHub webhook events into Go types. If not
	//supplied, MinimalEventDecoder is used.
	EventDecoder EventDecoder
	//A callback that gets called once per valid event received. The event
	//argument can have any type that can be returned by the Handler's
	//EventDecoder.
	Callback func(guid string, event Event)
}

//ServeHTTP implements the http.Handler interface.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	//check request method
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//protect against maliciously large payloads (GitHub payloads are capped at 25 MiB)
	bodyReader := io.LimitReader(r.Body, 25<<20)
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//check signature
	err = h.checkGitHubSignature(r, body)
	if err == errNoSignature {
		err = h.checkGiteaSignature(r, body)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	//decode event
	eventType := r.Header.Get("X-GitHub-Event")
	eventDecoder := EventDecoder(MinimalEventDecoder)
	if h.EventDecoder != nil {
		eventDecoder = h.EventDecoder
	}
	event, err := eventDecoder(eventType, []byte(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if event == nil {
		http.Error(w, "event type not supported", http.StatusNotImplemented)
		return
	}

	h.Callback(r.Header.Get("X-GitHub-Delivery"), event)
	w.WriteHeader(http.StatusNoContent)
}

var (
	errNoSignature      = errors.New("missing signature header (X-Hub-Signature or X-Gitea-Signature)")
	errInvalidSignature = errors.New("invalid signature header")
)

func (h Handler) checkGitHubSignature(r *http.Request, body []byte) error {
	signature := strings.TrimSpace(r.Header.Get("X-Hub-Signature"))
	if signature == "" {
		return errNoSignature
	}
	if len(signature) != 45 { // 40 hex digits plus "sha1=" prefix
		return errInvalidSignature
	}

	mac := hmac.New(sha1.New, []byte(h.SecretKey))
	mac.Write(body)
	expectedSignature := "sha1=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errInvalidSignature
	}

	return nil
}

func (h Handler) checkGiteaSignature(r *http.Request, body []byte) error {
	signature := strings.TrimSpace(r.Header.Get("X-Gitea-Signature"))
	if signature == "" {
		return errNoSignature
	}
	if len(signature) != 64 { // 64 hex digits, without any prefix
		return errInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(h.SecretKey))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errInvalidSignature
	}

	return nil
}
