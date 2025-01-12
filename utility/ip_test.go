// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utility_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/utility"

	"github.com/stretchr/testify/assert"
)

const (
	headerForwarded             = "Forwarded"
	headerXForwardedFor         = "X-Forwarded-For"
	headerXRealIP               = "X-Real-IP"
	headerXOriginalForwardedFor = "X-Original-Forwarded-For"
)

func TestExtractIP(t *testing.T) {
	assert.Nil(t, utility.ExtractIP(&http.Request{}))
	assert.Nil(t, utility.ExtractIP(&http.Request{RemoteAddr: "invalid"}))
	assert.Equal(t, "::1", utility.ExtractIP(&http.Request{RemoteAddr: "[::1]:1234"}).String())
	assert.Equal(t, "192.168.0.1", utility.ExtractIP(&http.Request{RemoteAddr: "192.168.0.1"}).String())

	req := &http.Request{
		RemoteAddr: "[::1]:1234",
		Header:     make(http.Header),
	}
	req.Header.Set(headerXForwardedFor, "client.invalid")
	assert.Equal(t, "::1", utility.ExtractIP(req).String())

	req.Header.Set(headerXRealIP, "127.1.2.3")
	assert.Equal(t, "127.1.2.3", utility.ExtractIP(req).String())

	req.Header.Set(headerForwarded, "for=_secret")
	assert.Equal(t, "127.1.2.3", utility.ExtractIP(req).String())

	req.Header.Set(headerForwarded, "for=[2001:db8:cafe::17]:4711")
	assert.Equal(t, "2001:db8:cafe::17", utility.ExtractIP(req).String())
}

func TestExtractIPRUM(t *testing.T) {
	assert.Nil(t, utility.ExtractIPRUM(&http.Request{}))
	assert.Nil(t, utility.ExtractIPRUM(&http.Request{RemoteAddr: "invalid"}))

	//	Direct
	assert.Equal(t, "::1", utility.ExtractIPRUM(&http.Request{RemoteAddr: "[::1]:1234"}).String())
	assert.Equal(t, "192.168.0.1", utility.ExtractIPRUM(&http.Request{RemoteAddr: "192.168.0.1"}).String())

	req := &http.Request{
		RemoteAddr: "[::1]:1234",
		Header:     make(http.Header),
	}

	//	X-Forwarded-For
	req.Header.Set(headerXForwardedFor, "client.invalid")
	assert.Equal(t, "::1", utility.ExtractIPRUM(req).String())

	//	X-Real-IP should override X-Forwarded-For
	req.Header.Set(headerXRealIP, "127.1.2.3")
	assert.Equal(t, "127.1.2.3", utility.ExtractIPRUM(req).String())

	//	Forwarded header should override X-Forwarded-For and X-Real-IP
	req.Header.Set(headerForwarded, "for=_secret") //	invalid, fall back
	assert.Equal(t, "127.1.2.3", utility.ExtractIPRUM(req).String())

	req.Header.Set(headerForwarded, "for=[2001:db8:cafe::17]:4711")
	assert.Equal(t, "2001:db8:cafe::17", utility.ExtractIPRUM(req).String())

	//	X-Original-Forwarded-For should override all others
	req.Header.Set(headerXOriginalForwardedFor, "38.60.220.1")
	assert.Equal(t, "38.60.220.1", utility.ExtractIPRUM(req).String())
}

func TestExtractIPFromHeader(t *testing.T) {
	for name, tc := range map[string]struct {
		header map[string]string
		ip     string
	}{
		"no header":                   {},
		"Invalid-X-Forwarded-For":     {header: map[string]string{headerXForwardedFor: "client.invalid"}},
		"Invalid-X-Real-IP-Invalid":   {header: map[string]string{headerXRealIP: "client.invalid"}},
		"Invalid-Forwarded":           {header: map[string]string{headerForwarded: "for=client.invalid"}},
		"Invalid-ForwardedMissingFor": {header: map[string]string{headerForwarded: "128.0.0.5"}},
		"X-Forwarded-For": {
			header: map[string]string{headerXForwardedFor: "123.0.0.1"},
			ip:     "123.0.0.1"},
		"X-Forwarded-For-First": {
			header: map[string]string{headerXForwardedFor: "123.0.0.1, 127.0.0.1"},
			ip:     "123.0.0.1"},
		"X-Real-IP": {
			header: map[string]string{headerXRealIP: "123.0.0.1:6060"},
			ip:     "123.0.0.1"},
		"X-Real-IP-Fallback": {
			header: map[string]string{headerXRealIP: "invalid", headerXForwardedFor: "182.0.0.9"},
			ip:     "182.0.0.9"},
		"Forwarded": {
			header: map[string]string{headerForwarded: "for=[2001:db8:cafe::17]:4711"},
			ip:     "2001:db8:cafe::17"},
		"Forwarded-Fallback": {
			header: map[string]string{headerForwarded: "invalid", headerXForwardedFor: "182.0.0.9"},
			ip:     "182.0.0.9"},
		"Forwarded-Fallback2": {
			header: map[string]string{headerForwarded: "invalid", headerXRealIP: "182.0.0.9"},
			ip:     "182.0.0.9"},
	} {
		t.Run("invalid"+name, func(t *testing.T) {
			header := make(http.Header)
			for k, v := range tc.header {
				header.Set(k, v)
			}
			ip := utility.ExtractIPFromHeader(header)
			if tc.ip == "" {
				assert.Nil(t, ip)
			} else {
				require.NotNil(t, ip)
				assert.Equal(t, tc.ip, ip.String())
			}
		})
	}
}

func TestExtractIPFromHeaderRum(t *testing.T) {
	for name, tc := range map[string]struct {
		header map[string]string
		ip     string
	}{
		"no header":                        {},
		"Invalid-X-Forwarded-For":          {header: map[string]string{headerXForwardedFor: "client.invalid"}},
		"Invalid-X-Real-IP-Invalid":        {header: map[string]string{headerXRealIP: "client.invalid"}},
		"Invalid-Forwarded":                {header: map[string]string{headerForwarded: "for=client.invalid"}},
		"Invalid-X-Original-Forwarded-For": {header: map[string]string{headerXOriginalForwardedFor: "client.invalid"}},
		"Invalid-ForwardedMissingFor":      {header: map[string]string{headerForwarded: "128.0.0.5"}},
		"X-Forwarded-For": {
			header: map[string]string{headerXForwardedFor: "123.0.0.1"},
			ip:     "123.0.0.1"},
		"X-Forwarded-For-First": {
			header: map[string]string{headerXForwardedFor: "123.0.0.1, 127.0.0.1"},
			ip:     "123.0.0.1"},
		"X-Real-IP": {
			header: map[string]string{headerXRealIP: "123.0.0.1:6060"},
			ip:     "123.0.0.1"},
		"X-Real-IP-Fallback": {
			header: map[string]string{headerXRealIP: "invalid", headerXForwardedFor: "182.0.0.9"},
			ip:     "182.0.0.9"},
		"Forwarded": {
			header: map[string]string{headerForwarded: "for=[2001:db8:cafe::17]:4711"},
			ip:     "2001:db8:cafe::17"},
		"Forwarded-Fallback": {
			header: map[string]string{headerForwarded: "invalid", headerXForwardedFor: "182.0.0.9"},
			ip:     "182.0.0.9"},
		"Forwarded-Fallback2": {
			header: map[string]string{headerForwarded: "invalid", headerXRealIP: "182.0.0.9"},
			ip:     "182.0.0.9"},
		"X-Original-Forwarded-For": {
			header: map[string]string{headerXOriginalForwardedFor: "192.169.56.101"},
			ip:     "192.169.56.101"},
	} {
		t.Run("invalid"+name, func(t *testing.T) {
			header := make(http.Header)
			for k, v := range tc.header {
				header.Set(k, v)
			}
			ip := utility.ExtractIPFromHeaderRum(header)
			if tc.ip == "" {
				assert.Nil(t, ip)
			} else {
				require.NotNil(t, ip)
				assert.Equal(t, tc.ip, ip.String())
			}
		})
	}
}

func TestParseIP(t *testing.T) {
	for name, tc := range map[string]struct {
		inp  string
		host string
	}{
		"IPv4":         {inp: "192.0.0.1", host: "192.0.0.1"},
		"IPv4WithPort": {inp: "192.0.0.1:8080", host: "192.0.0.1"},
		"IPv6":         {inp: "2001:db8::68", host: "2001:db8::68"},
		"customPort":   {inp: "168.14.10.23:8", host: "168.14.10.23"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.host, utility.ParseIP(tc.inp).String())
		})
	}

	for name, inp := range map[string]string{
		"invalidIP": "192.0.01",
		"randomStr": "foo",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, utility.ParseIP(inp))
		})
	}
}

func BenchmarkExtractIP(b *testing.B) {
	remote := "10.11.12.13"
	remoteWithPort := remote + ":8080"
	realIp := "54.55.101.102"
	xForwardedFor := "54.56.103.104"
	xForwardedForMultiple := "54.56.103.104 , 54.57.105.106 , 54.58.107.108"
	forwardedFor := "56.103.54.106"
	invalid := "invalid"
	empty := ""

	testCases := []struct {
		want                            string
		remote, forward, real, xForward *string
	}{
		{forwardedFor, &remoteWithPort, &forwardedFor, nil, nil},
		{realIp, &remoteWithPort, nil, &realIp, nil},
		{realIp, &remoteWithPort, nil, &realIp, &xForwardedFor},
		{xForwardedFor, &remoteWithPort, nil, nil, &xForwardedFor},
		{xForwardedFor, &remoteWithPort, nil, nil, &xForwardedForMultiple},
		{remote, &remoteWithPort, nil, nil, nil},
		{remote, &remoteWithPort, nil, &empty, &empty},
		{remote, &remoteWithPort, &invalid, &invalid, &invalid},
		{empty, nil, nil, nil, nil},
		{empty, &empty, &empty, &empty, &empty},
	}

	nilOrString := func(v *string) string {
		if v == nil {

			return "nil"
		}
		return *v
	}
	for _, tc := range testCases {
		name := fmt.Sprintf("extractIP remote: %v, Forwarded: %v, X-Real-IP: %v, X-Forwarded-For: %v",
			nilOrString(tc.remote), nilOrString(tc.forward), nilOrString(tc.real), nilOrString(tc.xForward))
		req := testRequest(tc.remote, tc.forward, tc.real, tc.xForward)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				utility.ExtractIP(req)
			}
		})
	}
}

func testRequest(remote, forward, real, xForward *string) *http.Request {
	req, _ := http.NewRequest("POST", "_", nil)
	if remote != nil {
		req.RemoteAddr = *remote
	}
	if forward != nil {
		req.Header.Add(headerForwarded, *forward)
	}
	if real != nil {
		req.Header.Add(headerXRealIP, *real)
	}
	if xForward != nil {
		req.Header.Add(headerXForwardedFor, *xForward)
	}
	return req
}
