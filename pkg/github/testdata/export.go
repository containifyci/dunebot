package testdata

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"
)

type SavedResponse struct {
	Url     string            `yaml:"url"`
	Urls    []string          `yaml:"urls"`
	Query   string            `yaml:"query"`
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
	Binary  bool
}

func (r *SavedResponse) Response(req *http.Request) *http.Response {
	header := make(http.Header)
	for k, v := range r.Headers {
		header.Add(k, v)
	}

	var body []byte
	if r.Binary {
		b, err := base64.StdEncoding.DecodeString(r.Body)
		if err != nil {
			panic("invalid base64 encoded binary body")
		}
		body = b
	} else {
		body = []byte(r.Body)
	}

	return &http.Response{
		Status:     http.StatusText(r.Status),
		StatusCode: r.Status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,

		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),

		Request: req,
	}
}

type ResponseHandlerFunc func(req *http.Request) *http.Response

type ResponseHandlerFuncWrapper struct {
	f      ResponseHandlerFunc
	Called bool
}

func NewResponseHandler(f ResponseHandlerFunc) *ResponseHandlerFuncWrapper {
	return &ResponseHandlerFuncWrapper{f: f}
}


func (f *ResponseHandlerFuncWrapper) Response(req *http.Request) *http.Response {
	f.Called = true
	return f.f(req)
}

type ResponseHandler interface {
	Response(req *http.Request) *http.Response
}

type Rule struct {
	Path  string
	Query string
	Count int

	responses        []SavedResponse
	responseHandlers []ResponseHandler
	err              error
}

type ResponsePlayer struct {
	Rules []*Rule
}

func NewResponsePlayer(dir string) *ResponsePlayer {
	rp := &ResponsePlayer{
		Rules: make([]*Rule, 0),
	}
	rp.AddRules(dir)
	return rp
}

func (rp *ResponsePlayer) AddRules(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		rp.AddUrlRule(filepath.Join(dir, entry.Name()))
	}
}

func (rp *ResponsePlayer) ParseRule(file string) Rule {
	rule := Rule{}

	d, err := os.ReadFile(file)
	if err != nil {
		rule.err = fmt.Errorf("failed to read response file: %s %e", file, err)
		return rule
	}

	if err := yaml.Unmarshal(d, &rule.responses); err != nil {
		rule.err = fmt.Errorf("failed to unmarshal response file: %s %e", file, err)
		return rule
	}

	return rule
}

func (rp *ResponsePlayer) AddRule(path, query, file string) *Rule {
	rule := rp.ParseRule(file)
	rule.Query = query
	rule.Path = path
	rp.Rules = append(rp.Rules, &rule)
	return &rule
}

func (rp *ResponsePlayer) ReplaceRule(path, query, file string) *Rule {
	rule := rp.ParseRule(file)
	rule.Query = query
	rule.Path = path
	for i, r := range rp.Rules {
		if r.Path == path && r.Query == query {
			rp.Rules[i] = &rule
		}
	}
	return &rule
}

func (rp *ResponsePlayer) AddUrlRule(file string) *Rule {
	rule := rp.ParseRule(file)
	if len(rule.responses) <= 0 {
		rule.err = fmt.Errorf("no responses found in file: %s", file)
		return &rule
	}
	if rule.responses[0].Query != "" {
		rule.Query = rule.responses[0].Query
	}
	if rule.responses[0].Url != "" {
		rule.Path = rule.responses[0].Url
		rp.Rules = append(rp.Rules, &rule)
	}
	for _, url := range rule.responses[0].Urls {
		r := rule
		r.Path = url
		rp.Rules = append(rp.Rules, &r)
	}

	return &rule
}

func (rp *ResponsePlayer) AddDynamicRule(path, query string, handler ResponseHandler) *Rule {
	rule := &Rule{Path: path, Query: query}
	rule.responseHandlers = make([]ResponseHandler, 0)
	rule.responseHandlers = append(rule.responseHandlers, handler)
	rp.Rules = append(rp.Rules, rule)
	return rule
}

func (rp *ResponsePlayer) SetDynamicRule(path, query string, handler ResponseHandler) *Rule {
	idx, rule := rp.findMatchRule(path, query)
	if rule != nil {
		rp.Rules = slices.Delete(rp.Rules, idx, idx+1)
	}
	rule = &Rule{Path: path, Query: query}
	rule.responseHandlers = append(make([]ResponseHandler, 0), handler)
	rp.Rules = append(rp.Rules, rule)
	return rule
}

func (r *Rule) AddResponseHandler(handler ...ResponseHandler) *Rule {
	r.responseHandlers = append(r.responseHandlers, handler...)
	return r
}

func errorResponse(req *http.Request, code int, msg string) (*http.Response, error) {
	body := strings.NewReader(msg)

	return &http.Response{
		Status:     http.StatusText(code),
		StatusCode: code,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,

		Header:        make(http.Header),
		Body:          io.NopCloser(body),
		ContentLength: body.Size(),

		Request: req,
	}, nil
}

func (rp *ResponsePlayer) findMatchRule(url, query string) (int, *Rule) {
	for i, rule := range rp.Rules {
		if rule.Query != "" && rule.Query == query &&
			rule.Path == url {
			return i, rule
		}
		if rule.Query == "" && rule.Path == url{
			return i, rule
		}
	}
	return 0, nil
}

func (rp *ResponsePlayer) findMatch(req *http.Request) *Rule {
	_, rule := rp.findMatchRule(req.URL.Path, req.URL.RawQuery)
	return rule
}

func (rp *ResponsePlayer) RoundTrip(req *http.Request) (*http.Response, error) {
	rule := rp.findMatch(req)
	if rule == nil {
		return errorResponse(req, http.StatusGone, fmt.Sprintf("no matching rule for \"%s %s\"", req.Method, req.URL.Path))
	}

	// report any error encountered during loading
	if rule.err != nil {
		return nil, rule.err
	}

	if rule.responseHandlers != nil {
		index := rule.Count % len(rule.responseHandlers)
		rule.Count++

		return rule.responseHandlers[index].Response(req), nil
	}

	// fail if there are no responses
	if len(rule.responses) == 0 {
		return errorResponse(req, http.StatusGone, fmt.Sprintf("no responses for \"%s %s\"", req.Method, req.URL.Path))
	}

	index := rule.Count % len(rule.responses)
	rule.Count++

	return rule.responses[index].Response(req), nil
}
