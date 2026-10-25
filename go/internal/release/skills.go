package release

import (
	"fmt"
	"os"
	"path/filepath"
)

// StageSkills gathers the skills each command keeps beside its code, as
// go/cmd/<command>/skills/<name>/, into out/skills/<name>/. A release bundle
// ships that one flat folder, so an agent links every skill by its name.
func (c Config) StageSkills() error {
	staged := filepath.Join(c.Out, "skills")
	if err := os.RemoveAll(staged); err != nil {
		return err
	}
	skills, err := filepath.Glob(filepath.Join(c.Root, "go", "cmd", "*", "skills", "*", "SKILL.md"))
	if err != nil {
		return err
	}
	owners := map[string]string{}
	for _, skill := range skills {
		if err := stageSkill(staged, filepath.Dir(skill), owners); err != nil {
			return err
		}
	}
	return nil
}

// stageSkill copies one skill folder into the staged folder. Two commands that
// keep a skill of the same name fail, since one would hide the other.
func stageSkill(staged, folder string, owners map[string]string) error {
	name := filepath.Base(folder)
	if owner, ok := owners[name]; ok {
		return fmt.Errorf("skill %s is kept in both %s and %s; rename one", name, owner, folder)
	}
	owners[name] = folder
	return os.CopyFS(filepath.Join(staged, name), os.DirFS(folder))
}
