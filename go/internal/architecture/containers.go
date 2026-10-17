package architecture

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Runtime is one product the repository ships and the container that owns
// it: the container whose claims cover the product's folder or whose name
// is the product's name.
type Runtime struct {
	runtimes.Product
	Owner string
}

// Containers is the containers view compared with what the repository
// ships. Files counts the container views read.
type Containers struct {
	Runtimes []Runtime
	Errors   []string
	Files    int
}

// Containers matches each owned container to a build target, entry point,
// or deploy file, and each product to one container. External systems need
// no evidence; a platform service the code uses needs an external system.
func (m *Model) Containers() Containers {
	c := &containers{m: m}
	for _, e := range m.Elements {
		c.element(e)
	}
	report := runtimes.Find(m.repo.root, m.repo.sorted(), m.product, readers)
	for _, p := range report.Products {
		c.product(p)
	}
	c.containersWithoutRuntime()
	c.services(report.Services)
	c.Errors = append(c.Errors, report.Errors...)
	c.Files = m.viewCount("containers")
	return c.Containers
}

type containers struct {
	Containers
	m         *Model
	owned     []Element
	externals []Element
	matched   map[string]bool
}

func (c *containers) element(e Element) {
	if e.View != "containers" {
		return
	}
	switch {
	case strings.HasSuffix(e.Kind, "_Ext"):
		c.externals = append(c.externals, e)
	case internal(e.Kind, "Container"):
		c.owned = append(c.owned, e)
	}
}

func (c *containers) fail(at Location, format string, args ...any) {
	c.Errors = append(c.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

// product records a product's one owner, or fails a product nobody or two
// containers claim.
func (c *containers) product(p runtimes.Product) {
	owners := c.owners(p)
	at := Location(p.At)
	switch len(owners) {
	case 0:
		c.fail(at, "%s %s has no container in the containers view", p.Kind, p.Name)
	case 1:
		c.own(owners[0], p)
	default:
		c.fail(at, "%s %s is claimed by containers %s at %s and %s at %s", p.Kind, p.Name, owners[0].ID, owners[0].At, owners[1].ID, owners[1].At)
	}
}

func (c *containers) own(owner Element, p runtimes.Product) {
	if c.matched == nil {
		c.matched = map[string]bool{}
	}
	c.matched[owner.ID] = true
	c.Runtimes = append(c.Runtimes, Runtime{Product: p, Owner: owner.ID})
	if !namesTechnology(owner.Technology, p.Technology) {
		c.fail(owner.At, "container %s says %q but %s %s at %s is %s", owner.ID, owner.Technology, p.Kind, p.Name, p.At, p.Technology)
	}
}

func (c *containers) owners(p runtimes.Product) []Element {
	var owners []Element
	for _, e := range c.owned {
		if strings.EqualFold(e.Name, p.Name) || c.claims(e.ID, p.Paths) || c.names(e.ID, p) {
			owners = append(owners, e)
		}
	}
	return owners
}

// names reports whether owner's text names a package product. A code
// span or label may name the package or a path covering an entry point,
// such as dist/esm for dist/esm/index.js; a description word must be an
// entry point itself. Entry points are often build output nobody tracks.
func (c *containers) names(owner string, p runtimes.Product) bool {
	if !p.Package {
		return false
	}
	for _, mention := range c.m.Mentions {
		if mention.Owner == owner && namesPackage(mention, p) {
			return true
		}
	}
	return false
}

func namesPackage(m Mention, p runtimes.Product) bool {
	if !m.Named {
		return contains(p.Entries, m.Text)
	}
	return strings.EqualFold(m.Text, p.Name) || coversAny(m.Text, p.Entries)
}

// claims reports whether owner's claimed paths cover any of paths.
func (c *containers) claims(owner string, paths []string) bool {
	for _, claim := range c.m.Claims {
		if claim.Owner == owner && coversAny(claim.Path, paths) {
			return true
		}
	}
	return false
}

func coversAny(claim string, paths []string) bool {
	for _, p := range paths {
		if covers(claim, p) {
			return true
		}
	}
	return false
}

// containersWithoutRuntime fails each owned container no product matched,
// and each owned store that names no path.
func (c *containers) containersWithoutRuntime() {
	for _, e := range c.owned {
		if e.Kind != "Container" {
			c.store(e)
			continue
		}
		if !c.matched[e.ID] {
			c.noRuntime(e)
		}
	}
}

func (c *containers) noRuntime(e Element) {
	if claimed := c.claimed(e.ID); len(claimed) > 0 {
		c.fail(e.At, "container %s claims only source layers (%s): no build target, entry point, or deploy file is under them; show them in the component view", e.ID, strings.Join(claimed, ", "))
		return
	}
	c.fail(e.At, "container %s matches no build target, entry point, or deploy file by name", e.ID)
}

// claimed lists the distinct paths owner names in any view, since one ID
// is one element across views.
func (c *containers) claimed(owner string) []string {
	var paths []string
	for _, claim := range c.m.Claims {
		if claim.Owner == owner && !contains(paths, claim.Path) {
			paths = append(paths, claim.Path)
		}
	}
	return paths
}

func (c *containers) store(e Element) {
	if len(c.claimed(e.ID)) == 0 {
		c.fail(e.At, "store %s names no schema, configuration, or source path", e.ID)
	}
}

// services fails each platform service the code uses that no external
// system in the containers view names.
func (c *containers) services(services []runtimes.Service) {
	for _, s := range services {
		if !c.external(s.Name) {
			c.fail(Location(s.At), "%s is used here but no external system in the containers view names it", s.Name)
		}
	}
}

// external reports an external system whose name, technology, or
// description names the service.
func (c *containers) external(name string) bool {
	for _, e := range c.externals {
		if containsFold(e.Name+" "+e.Technology+" "+e.Description, name) {
			return true
		}
	}
	return false
}

func containsFold(text, name string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(name))
}
