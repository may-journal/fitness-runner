// Package requirements proves the Go module's docs/requirements as a user
// meets them. Each test runs the installer at FITNESS_INSTALL in a copied
// example repo.
package requirements

import (
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/acceptance"
)

func TestMain(m *testing.M) {
	acceptance.Main(m)
}

func Test0001_1(t *testing.T) {
	acceptance.Cases{
		{"t.Run cases, nested ones, a named case table, an anonymous one and rows without a name: the check sees each case name, then its location, then the step a name opens with", func(t *testing.T) {
			t.Parallel()
			suite := "package cases\n\nimport \"testing\"\n\n" +
				"type edgeCase struct {\n\tname string\n\trun  func(t *testing.T)\n}\n\n" +
				"type table []struct {\n\tName string\n\twant int\n}\n\n" +
				"type empty struct{}\n\ntype listed struct {\n\tnames []string\n}\n\n" +
				"func Test0001_1(t *testing.T) {\n" +
				"\tt.Run(\"0001.1.1 installs\", func(t *testing.T) {\n\t\tt.Run(\"nested\", func(t *testing.T) {})\n\t})\n" +
				"\trunCases(t, []edgeCase{\n\t\t{\"0001.1.2: updates\", nil},\n\t\t{name: \"keyed\"},\n\t\t{run: nil},\n\t\t{},\n\t\tshared,\n\t})\n" +
				"\t_ = table{{`0001.1.10 raw`, 1}}\n" +
				"\t_ = []struct{ in, want string }{{\"not a case\", \"x\"}}\n" +
				"\t_ = []empty{{}}\n\t_ = []listed{{nil}}\n\t_ = []int{1}\n\t_ = map[string]int{}\n" +
				"\tt.Run(name, nil)\n\tt.Run(\"three\", nil, nil)\n\thelper(\"not a case\")\n" +
				"\t_ = []edgeCase{{0, nil}}\n" +
				"\ttype local struct{ name string }\n\tvar count int\n\t_ = []local{{\"0001.1.6 local\"}}\n" +
				"\t_ = []*edgeCase{{\"0001.1.7 pointer\", nil}, &edgeCase{name: \"addressed\"}}\n" +
				"\t_ = map[string]struct{ want int }{\"0001.1.8 mapped\": {1}}\n\t_ = map[int]edgeCase{}\n" +
				"\tt.Run(\"0001.1.4, then\", nil)\n\tt.Run(\"0001.1.5.1 deeper\", nil)\n\tt.Run(\"0001.1.9.\", nil)\n" +
				"}\n\nfunc TestHelper(t *testing.T) {\n\tt.Run(\"0001.1.3 outside\", nil)\n}\n"
			out := acceptance.SharedScan(t, map[string]string{"cases/0001_test.go": suite})
			acceptance.Wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1","Loc":"cases/0001_test.go:22"},`+
				`{"Name":"nested","Loc":"cases/0001_test.go:23"},`+
				`{"Name":"0001.1.2: updates","Step":"0001.1.2","Loc":"cases/0001_test.go:26"},`+
				`{"Name":"keyed","Loc":"cases/0001_test.go:27"},`+
				`{"Name":"0001.1.10 raw","Step":"0001.1.10","Loc":"cases/0001_test.go:32"},`+
				`{"Name":"0001.1.6 local","Step":"0001.1.6","Loc":"cases/0001_test.go:44"},`+
				`{"Name":"0001.1.7 pointer","Step":"0001.1.7","Loc":"cases/0001_test.go:45"},`+
				`{"Name":"addressed","Loc":"cases/0001_test.go:45"},`+
				`{"Name":"0001.1.8 mapped","Step":"0001.1.8","Loc":"cases/0001_test.go:46"},`+
				`{"Name":"0001.1.4, then","Step":"0001.1.4","Loc":"cases/0001_test.go:48"},`+
				`{"Name":"0001.1.5.1 deeper","Loc":"cases/0001_test.go:49"},`+
				`{"Name":"0001.1.9.","Step":"0001.1.9","Loc":"cases/0001_test.go:50"}],"Loc":"cases/0001_test.go:21"`,
				`"Name":"TestHelper","ID":"","Loc"`)
		}},
	}.Run(t)
}
