package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// commentEndpoint matches the REST paths that post or edit comments on
// issues, PR lines, replies, reviews and commits.
var commentEndpoint = regexp.MustCompile(`^/?repos/[^/]+/[^/]+/(issues/[^/]+/comments|issues/comments/[^/]+|pulls/[^/]+/comments|pulls/comments/[^/]+(/replies)?|pulls/[^/]+/reviews(/[^/]+(/events)?)?|commits/[^/]+/comments|comments/[^/]+)$`)

// commentMutation matches the GraphQL mutations that add or edit a comment,
// a review, a review thread or reply, or a discussion comment.
var commentMutation = regexp.MustCompile(`\b(addComment|updateIssueComment|addPullRequestReview|addPullRequestReviewComment|addPullRequestReviewThread|addPullRequestReviewThreadReply|updatePullRequestReviewComment|updatePullRequestReview|submitPullRequestReview|addDiscussionComment|updateDiscussionComment)\b`)

// apiValueFlags are the gh api flags that take a value Fitness does not read.
var apiValueFlags = []string{"-H", "--header", "-q", "--jq", "-t", "--template", "-p", "--preview", "--hostname", "--cache"}

// apiFieldFlags give one key=value field: raw ones as text, typed ones with
// @file for a file's text.
var (
	apiRawFields   = []string{"-f", "--raw-field"}
	apiTypedFields = []string{"-F", "--field"}
)

// reviewCommentKey is the field gh api sends each review line comment's
// body in.
const reviewCommentKey = "comments[][body]"

// apiCall is what a gh api call sends: its endpoint, method, fields in
// order, and any --input text.
type apiCall struct {
	endpoint, method string
	fields           []apiField
	input            *shellWord
}

// apiField is one field a gh api call sends.
type apiField struct {
	key   string
	value shellWord
}

// apiPost reads the comment a gh api call posts or edits, from the words
// after api. A call that reads, or whose body is unknown, posts nothing to
// judge.
func apiPost(args []shellWord, call ghCall) (ghPost, bool) {
	api, ok := readAPICall(args, call)
	if !ok || strings.EqualFold(api.method, "GET") {
		return ghPost{}, false
	}
	parts, ok := api.parts()
	parts = slices.DeleteFunc(parts, emptyPart)
	return ghPost{comment: true, parts: parts}, ok && len(parts) > 0
}

// readAPICall reads a gh api call's words; false when one it needs is unknown.
func readAPICall(args []shellWord, call ghCall) (apiCall, bool) {
	r := apiReader{call: call, ok: true}
	for ; r.i < len(args) && r.ok; r.i++ {
		r.read(args)
	}
	return r.api, r.ok && r.api.endpoint != ""
}

// apiReader walks a gh api call's words once.
type apiReader struct {
	call ghCall
	api  apiCall
	i    int
	ok   bool
}

// read takes the word at the cursor, and its value when the flag has one.
func (r *apiReader) read(args []shellWord) {
	word := args[r.i]
	name, value, attached := splitAPIFlag(word.text)
	if !attached && r.takesValue(name) {
		r.i++
		word = wordAt(args, r.i)
		value = word.text
	}
	r.apply(name, value, word)
}

// takesValue reports whether a flag reads the next word as its value.
func (r *apiReader) takesValue(name string) bool {
	_, handled := apiFlagReaders[name]
	return handled || slices.Contains(apiValueFlags, name)
}

// apply records one flag or the endpoint. A field may be unknown, since
// only a body field's text matters; any other unknown word stops the read.
func (r *apiReader) apply(name, value string, word shellWord) {
	isField := slices.Contains(apiRawFields, name) || slices.Contains(apiTypedFields, name)
	if !word.known && !isField {
		r.ok = false
		return
	}
	if read, ok := apiFlagReaders[name]; ok {
		read(r, name, value, word.known)
		return
	}
	r.endpoint(name)
}

// apiFlagReaders record the gh api flags whose values Fitness reads.
var apiFlagReaders = map[string]func(r *apiReader, name, value string, known bool){}

func init() {
	method := func(r *apiReader, _, value string, _ bool) { r.api.method = value }
	input := func(r *apiReader, _, value string, _ bool) { r.api.input = r.fileWord(value) }
	for _, name := range []string{"-X", "--method"} {
		apiFlagReaders[name] = method
	}
	apiFlagReaders["--input"] = input
	for _, name := range append(slices.Clone(apiRawFields), apiTypedFields...) {
		apiFlagReaders[name] = (*apiReader).field
	}
}

// apiHost is the URL prefix a gh api endpoint may carry.
const apiHost = "https://api.github.com/"

// endpoint records the first word that is no flag as the endpoint, without
// its query string.
func (r *apiReader) endpoint(word string) {
	if !strings.HasPrefix(word, "-") && r.api.endpoint == "" {
		r.api.endpoint = strings.TrimPrefix(strings.SplitN(word, "?", 2)[0], apiHost)
	}
}

// field records one key=value field. A typed @file value reads the file,
// unless the shell builds its path.
func (r *apiReader) field(name, value string, known bool) {
	key, text, _ := strings.Cut(value, "=")
	word := shellWord{text: text, known: known}
	if known && slices.Contains(apiTypedFields, name) && strings.HasPrefix(text, "@") {
		word = *r.fileWord(strings.TrimPrefix(text, "@"))
	}
	r.api.fields = append(r.api.fields, apiField{key: key, value: word})
}

// fileWord reads a file a flag names, or stdin for -, as a word.
func (r *apiReader) fileWord(path string) *shellWord {
	text, ok := r.call.bodyFile(path)
	return &shellWord{text: text, known: ok}
}

// splitAPIFlag splits --flag=value, and a short flag with its value attached
// right after it, into the flag and that value.
func splitAPIFlag(word string) (name, value string, attached bool) {
	if strings.HasPrefix(word, "--") {
		return strings.Cut(word, "=")
	}
	if len(word) > 2 && word[0] == '-' {
		return word[:2], word[2:], true
	}
	return word, "", false
}

// wordAt is args[i], or an unknown word past the end.
func wordAt(args []shellWord, i int) shellWord {
	if i < len(args) {
		return args[i]
	}
	return shellWord{}
}

// parts names each body the call posts, or false when it is no comment call
// or a body it sends is unknown.
func (api apiCall) parts() ([]postPart, bool) {
	if api.endpoint == "graphql" {
		return api.graphQLParts()
	}
	if !commentEndpoint.MatchString(api.endpoint) {
		return nil, false
	}
	if api.input != nil {
		return inputParts(*api.input)
	}
	bodies, bodiesKnown := api.values("body")
	lines, linesKnown := api.values(reviewCommentKey)
	return composeParts(bodies, lines), bodiesKnown && linesKnown
}

// values are the texts of every field named key, or false when one is
// unknown.
func (api apiCall) values(key string) ([]string, bool) {
	var texts []string
	for _, f := range api.fields {
		if f.key != key {
			continue
		}
		if !f.value.known {
			return nil, false
		}
		texts = append(texts, f.value.text)
	}
	return texts, true
}

// composeParts names a call's comment bodies, then its review line comments
// by their place.
func composeParts(bodies, lines []string) []postPart {
	var parts []postPart
	for _, body := range bodies {
		parts = append(parts, postPart{subject: commentSubject, text: body})
	}
	for i, line := range lines {
		parts = append(parts, postPart{subject: fmt.Sprintf("review line comment %d", i+1), text: line})
	}
	return parts
}

// commentSubject names a call's own comment body, apart from its review
// line comments.
const commentSubject = "the comment body"

// emptyPart reports whether a body holds no text, such as a review sent with
// line comments alone.
func emptyPart(part postPart) bool {
	return strings.TrimSpace(part.text) == ""
}

// apiInput is the part of a --input JSON body that holds comment text.
type apiInput struct {
	Body     *string `json:"body"`
	Comments []struct {
		Body string `json:"body"`
	} `json:"comments"`
}

// inputParts names the body and each review line comment of a --input
// JSON file.
func inputParts(input shellWord) ([]postPart, bool) {
	var body apiInput
	if !input.known || json.Unmarshal([]byte(input.text), &body) != nil {
		return nil, false
	}
	var bodies, lines []string
	if body.Body != nil {
		bodies = []string{*body.Body}
	}
	for _, c := range body.Comments {
		lines = append(lines, c.Body)
	}
	return composeParts(bodies, lines), true
}

// bodyVariable matches a GraphQL body argument given as a variable,
// capturing the variable's name.
var bodyVariable = regexp.MustCompile(`\bbody\s*:\s*\$(\w+)`)

// graphQLParts reads the body variables of a comment mutation. A body
// written inside the query is not read, so that call is not judged.
func (api apiCall) graphQLParts() ([]postPart, bool) {
	query, vars, ok := api.graphQLRequest()
	if !ok || !commentMutation.MatchString(query) {
		return nil, false
	}
	bodies, ok := bodyValues(bodyVariable.FindAllStringSubmatch(query, -1), vars)
	return composeParts(bodies, nil), ok
}

// bodyValues reads the variable each body argument names; false when there
// is none, or one is unknown.
func bodyValues(names [][]string, vars func(string) (string, bool)) ([]string, bool) {
	var bodies []string
	for _, m := range names {
		text, ok := vars(m[1])
		if !ok {
			return nil, false
		}
		bodies = append(bodies, text)
	}
	return bodies, len(bodies) > 0
}

// graphQLRequest is a GraphQL call's query and a reader of its variables,
// from --input JSON or from fields.
func (api apiCall) graphQLRequest() (string, func(string) (string, bool), bool) {
	if api.input != nil {
		return inputGraphQL(*api.input)
	}
	query, ok := api.values("query")
	vars := func(name string) (string, bool) {
		texts, known := api.values(name)
		return strings.Join(texts, ""), known && len(texts) == 1
	}
	return strings.Join(query, ""), vars, ok && len(query) == 1
}

// graphQLInput is a --input GraphQL request.
type graphQLInput struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// inputGraphQL reads a --input GraphQL request's query and variables.
func inputGraphQL(input shellWord) (string, func(string) (string, bool), bool) {
	var request graphQLInput
	ok := input.known && json.Unmarshal([]byte(input.text), &request) == nil
	vars := func(name string) (string, bool) {
		text, isText := request.Variables[name].(string)
		return text, isText
	}
	return request.Query, vars, ok
}
