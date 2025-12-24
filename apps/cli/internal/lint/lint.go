package lint

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

// Run validates curated palace artifacts using embedded schemas.
func Run(rootPath string) error {
	var problems []string
	palacePath := filepath.Join(rootPath, ".palace", "palace.jsonc")
	if err := validate.JSONC(palacePath, "palace"); err != nil {
		problems = append(problems, err.Error())
	}

	roomPaths, _ := filepath.Glob(filepath.Join(rootPath, ".palace", "rooms", "*.jsonc"))
	for _, p := range roomPaths {
		if err := validate.JSONC(p, "room"); err != nil {
			problems = append(problems, err.Error())
		}
	}

	playbookPaths, _ := filepath.Glob(filepath.Join(rootPath, ".palace", "playbooks", "*.jsonc"))
	for _, p := range playbookPaths {
		if err := validate.JSONC(p, "playbook"); err != nil {
			problems = append(problems, err.Error())
		}
	}

	profilePath := filepath.Join(rootPath, ".palace", "project-profile.json")
	if err := validate.JSON(profilePath, "project-profile"); err != nil {
		problems = append(problems, err.Error())
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}
