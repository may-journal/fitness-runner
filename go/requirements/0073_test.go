package requirements

import (
	"strings"
	"testing"
)

const (
	aaCheck   = "architecture-actors"
	acDoc     = "docs/requirements/0001-open.md"
	aaGiven   = acDoc + ":18: Given "
	aaContext = "architecture/01-system-context.md"
	aaRel     = "Rel(user, app, \"3\")"
	aaHost    = aaRel + "\nPerson(host, \"4 Host\")"
	aaOpens   = " opens the app"
)

// acDocWith is requirement 0001 with one acceptance whose Given is given.
// The Given sits on line 18.
func acDocWith(given string) string {
	return "# 0001 Open\n\n## Why\n\nI open the app.\n\n## Measurement\n\nopens\n-\ntries\n\nSource: #1\n\n## Requirements\n\n- 0001.1\n    - Given " + given + "\n        - When it starts\n            - Then it opens\n"
}

// aaFixture is the default architecture with its one Given replaced.
func aaFixture(given string) map[string]string {
	files := acFixture(acRoot)
	files[acDoc] = acDocWith(given)
	return files
}

// aaWith is aaFixture with one edit to the context diagram.
func aaWith(given, oldContext, newContext string) map[string]string {
	files := aaFixture(given)
	files[acRoot+acContext] = strings.Replace(files[acRoot+acContext], oldContext, newContext, 1)
	return files
}

func aaRun(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, aaCheck)
}

// aaNoActor is the failure for a Given whose text names no actor among
// the declared names.
func aaNoActor(given, where, actors string) string {
	return aaGiven + "\"" + given + "\"" + where + "; actors declared at " + aaContext + ":3 are " + actors
}

func Test0073_1(t *testing.T) {
	t.Parallel()
	out, code := aaRun(t, map[string]string{"src/main.go": "package main\n", acDoc: acDocWith("no network")})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, aaCheck)
	out, code = aaRun(t, aaFixture("no network"))
	sees(t, out, code, 1, aaNoActor("no network", " starts with no actor", "User"))
}

func Test0073_2(t *testing.T) {
	t.Parallel()
	for _, given := range []string{"User" + aaOpens, "user" + aaOpens, "User"} {
		out, code := aaRun(t, aaFixture(given))
		sees(t, out, code, 0)
		passedWith(t, out, aaCheck, "1")
	}
	for _, given := range []string{"Users open the app", "Admin" + aaOpens, "USER" + aaOpens} {
		out, code := aaRun(t, aaFixture(given))
		sees(t, out, code, 1, aaNoActor(given, " starts with no actor", "User"))
	}
}

func Test0073_3(t *testing.T) {
	t.Parallel()
	out, code := aaRun(t, aaFixture("User has an app"))
	sees(t, out, code, 0)
	given := "a user" + aaOpens
	out, code = aaRun(t, aaFixture(given))
	sees(t, out, code, 1, aaNoActor(given, " starts with no actor", "User"))
}

func Test0073_4(t *testing.T) {
	t.Parallel()
	given := "Host" + aaOpens
	out, code := aaRun(t, aaWith(given, aaRel, aaHost))
	sees(t, out, code, 0)
	repo := example(t, "happyRepo", aaFixture(given))
	for _, changed := range []string{"README.md", "src/store/store.go", acRoot + acContext} {
		out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, aaCheck)
		sees(t, out, code, 1, aaNoActor(given, " starts with no actor", "User"))
	}
}

func Test0073_5(t *testing.T) {
	t.Parallel()
	out, code := aaRun(t, aaWith("Host"+aaOpens, aaRel, aaHost))
	sees(t, out, code, 0)
	out, code = aaRun(t, aaWith("User"+aaOpens, aaRel, aaRel+"\nPerson(User, \"4 Host\")"))
	sees(t, out, code, 1, aaGiven+"actor \"User\" is ambiguous: user at "+aaContext+":5 and User at "+aaContext+":8")
}

func Test0073_6(t *testing.T) {
	t.Parallel()
	out, code := aaRun(t, aaWith("User and Host open the app", aaRel, aaHost))
	sees(t, out, code, 0)
	given := "User and Admin open the app"
	out, code = aaRun(t, aaWith(given, aaRel, aaHost))
	sees(t, out, code, 1, aaNoActor(given, " names no actor at \"Admin open the app\"", "User, Host"))
	out, code = aaRun(t, aaWith("User and", aaRel, aaHost))
	sees(t, out, code, 1, aaNoActor("User and", " names no actor at \"\"", "User, Host"))
}

func Test0073_8(t *testing.T) {
	t.Parallel()
	files := aaFixture("User" + aaOpens)
	files[acDoc] = strings.Replace(files[acDoc], "        - When it starts\n", "", 1)
	out, code := aaRun(t, files)
	sees(t, out, code, 0)
	passedWith(t, out, aaCheck, "1")
	if strings.Contains(out, acDoc) {
		t.Errorf("doc shape belongs to the requirements check, got:\n%s", out)
	}
}

func Test0073_7(t *testing.T) {
	t.Parallel()
	out, code := aaRun(t, aaFixture("User"+aaOpens))
	sees(t, out, code, 0)
	for _, given := range []string{"App" + aaOpens, "app" + aaOpens, "Person" + aaOpens, "p" + aaOpens} {
		out, code := aaRun(t, aaFixture(given))
		sees(t, out, code, 1, aaNoActor(given, " starts with no actor", "User"))
	}
}
