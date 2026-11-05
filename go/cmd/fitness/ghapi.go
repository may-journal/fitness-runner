package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// commentEndpoint matches the REST paths that post or edit an issue comment,
// a PR line comment or its reply, or a review.
var commentEndpoint = regexp.MustCompile(`^/?repos/[^/]+/[^/]+/(issues/[^/]+/comments|issues/comments/[^/]+|pulls/[^/]+/comments|pulls/comments/[^/]+(/replies)?|pulls/[^/]+/reviews(/[^/]+)?)$`)

// commentMutation matches the GraphQL mutations that add or edit a comment,
// a review or a review reply.
var commentMutation = regexp.MustCompile(`\b(addComment|updateIssueComment|addPullRequestReview|addPullRequestReviewComment|addPullRequestReviewThreadReply|updatePullRequestReviewComment|updatePullRequestReview|submitPullRequestReview)\b`)

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

// endpoint records the first word that is no flag as the endpoint, without
// its query string.
func (r *apiReader) endpoint(word string) {
	if !strings.HasPrefix(word, "-") && r.api.endpoint == "" {
		r.api.endpoint = strings.SplitN(word, "?", 2)[0]
	}
}

// field records one key=value field; a typed @file value reads the file.
func (r *apiReader) field(name, value string, known bool) {
	key, text, _ := strings.Cut(value, "=")
	word := shellWord{text: text, known: known}
	if slices.Contains(apiTypedFields, name) && strings.HasPrefix(text, "@") {
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
	return api.fieldParts("body")
}

// graphQLParts reads the body variable of a comment mutation. A body written
// inside the query is not read, so that call is not judged.
func (api apiCall) graphQLParts() ([]postPart, bool) {
	query, ok := api.field("query")
	if !ok || !commentMutation.MatchString(query) {
		return nil, false
	}
	return api.fieldParts("body")
}

// field is the text of the first field named key, when known.
func (api apiCall) field(key string) (string, bool) {
	for _, f := range api.fields {
		if f.key == key {
			return f.value.text, f.value.known
		}
	}
	return "", false
}

// fieldParts names the body field and each review line comment field.
func (api apiCall) fieldParts(key string) ([]postPart, bool) {
	var parts []postPart
	for _, f := range api.fields {
		part, ours := fieldPart(f, key, lineComments(parts))
		if ours && !f.value.known {
			return nil, false
		}
		if ours {
			parts = append(parts, part)
		}
	}
	return parts, true
}

// fieldPart names one field when it holds a body: the comment body, or the
// nth review line comment.
func fieldPart(f apiField, key string, n int) (postPart, bool) {
	switch f.key {
	case key:
		return postPart{subject: commentSubject, text: f.value.text}, true
	case reviewCommentKey:
		return postPart{subject: fmt.Sprintf("review line comment %d", n+1), text: f.value.text}, true
	}
	return postPart{}, false
}

// lineComments counts the review line comments among parts.
func lineComments(parts []postPart) int {
	n := 0
	for _, part := range parts {
		if part.subject != commentSubject {
			n++
		}
	}
	return n
}

// commentSubject names a call's own comment body, apart from its review
// line comments.
const commentSubject = "the comment body"

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
	var parts []postPart
	if body.Body != nil {
		parts = append(parts, postPart{subject: commentSubject, text: *body.Body})
	}
	for i, c := range body.Comments {
		parts = append(parts, postPart{subject: fmt.Sprintf("review line comment %d", i+1), text: c.Body})
	}
	return parts, true
}
