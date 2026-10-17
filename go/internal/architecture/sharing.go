package architecture

// share walks each runtime from the components its product folders are
// built from, along the observed edges, and fails a component declared
// under another runtime's container. A component outside every container
// boundary is shared and may be reached by any runtime.
func (c *components) share() {
	for _, rt := range c.m.Containers().Runtimes {
		c.runtime(rt)
	}
}

func (c *components) runtime(rt Runtime) {
	seen := map[string]bool{}
	for _, file := range sortedKeys(c.owners) {
		if coversAnyFolder(rt.Paths, file) {
			c.reach(rt, c.owners[file], Location(rt.At), true, seen)
		}
	}
}

func coversAnyFolder(folders []string, file string) bool {
	for _, folder := range folders {
		if covers(folder, file) {
			return true
		}
	}
	return false
}

// reach visits unit once per runtime, then follows its observed edges.
func (c *components) reach(rt Runtime, unit string, at Location, built bool, seen map[string]bool) {
	if seen[unit] {
		return
	}
	seen[unit] = true
	c.declared(rt, unit, at, built)
	for _, pair := range c.order {
		if pair[0] == unit {
			c.reach(rt, pair[1], Location(c.edges[pair].At), false, seen)
		}
	}
}

// declared fails a unit under a container boundary other than the
// runtime's own container.
func (c *components) declared(rt Runtime, unit string, at Location, built bool) {
	container := c.container(unit)
	if container == "" || container == rt.Owner {
		return
	}
	b := c.elements[container]
	if built {
		c.fail(at, "%s %s is built from %s, which is under container %s at %s, not %s", rt.Kind, rt.Name, unit, container, b.At, rt.Owner)
		return
	}
	c.fail(at, "%s %s reaches %s here, but %s is under container %s at %s, not %s; a component two runtimes share belongs outside every container boundary", rt.Kind, rt.Name, unit, unit, container, b.At, rt.Owner)
}

// container is the nearest boundary enclosing id whose ID is a container
// of the containers view, or empty for a shared component.
func (c *components) container(id string) string {
	for id = c.elements[id].Parent; id != ""; id = c.elements[id].Parent {
		if c.containers[id] {
			return id
		}
	}
	return ""
}
