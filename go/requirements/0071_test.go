package requirements

import "testing"

func Test0071_1(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, pairedRepo())
	write(t, repo, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "test(reqs): check sums twice")
	sees(t, out, code, 1, pairedTest+" changed without its requirement doc "+pairedDoc)
}

func Test0071_2(t *testing.T) {
	t.Parallel()
	out, code := prRun(t, pairedRepo(), map[string]string{pairedDoc: requirementDoc("0001.1") + "\n"}, "", "docs(requirements): reword 0001")
	sees(t, out, code, 1, pairedDoc+" changed without its requirement test "+pairedTest)
}

func Test0071_3(t *testing.T) {
	t.Parallel()
	doc := "docs/requirements/0002-adds-again.md"
	test := map[string]string{twoTest: edited(twoRepo()[twoTest])}
	out, code := stagedRun(t, twoRepo(), test)
	sees(t, out, code, 1, twoTest+" changed without its requirement doc "+pairedDoc+" or "+doc)
	out, code = stagedRun(t, twoRepo(), with(test, map[string]string{doc: twoRepo()[doc] + "\n"}))
	sees(t, out, code, 0)
}

func Test0071_4(t *testing.T) {
	t.Parallel()
	out, code := stagedRun(t, pairedRepo(), map[string]string{"reqs/tools_test.go": edited(pairedRepo()["reqs/tools_test.go"])})
	sees(t, out, code, 0)
}

func Test0071_5(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, pairedRepo())
	write(t, repo, map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))})
	user(t, repo, env, "add", "-A")
	out, code := user(t, repo, env, "commit", "-m", "test(reqs): check sums twice", "-m", exempt)
	sees(t, out, code, 0, "test(reqs): check sums twice")
}

func Test0071_6(t *testing.T) {
	t.Parallel()
	change := map[string]string{pairedTest: edited(testFile("reqs", "Test0001_1"))}
	out, code := prRun(t, pairedRepo(), change, "Rewords one test.", "test(reqs): reword", exempt)
	sees(t, out, code, 1, `"`+exempt+`" is not listed in the pull request body`)
	out, code = prRun(t, pairedRepo(), change, "Rewords one test.\n\n"+exempt+"\n", "test(reqs): reword", exempt)
	sees(t, out, code, 0)
}

func Test0071_7(t *testing.T) {
	t.Parallel()
	keep := []string{"--no-edit"}
	for _, r := range [][][]string{
		{{"merge", "--no-edit", "side"}, keep, {"--amend", "-m", "chore(reqs): merge the reword"}},
		{{"cherry-pick", "side"}, keep},
		{{"revert", "--no-edit", "side"}, {"-m", "revert(reqs): undo the reword"}},
	} {
		out, code := replayed(t, r[0], r[1:]...)
		sees(t, out, code, 0)
	}
}

func Test0071_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", pairedRepo())
	git(t, repo, "checkout", "-q", "-b", "base")
	commit(t, repo, map[string]string{pairedDoc: requirementDoc("0001.1") + "\n"}, "docs(requirements): reword 0001")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "work", "HEAD~1")
	commit(t, repo, map[string]string{"README.md": readme + "\nMore words.\n"}, "docs(readme): add words")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-edit", "base")
	out, code := fitness(t, repo, prEnv(t, ""), "requirements")
	sees(t, out, code, 0)
}
