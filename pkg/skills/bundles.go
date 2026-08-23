package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sipeed/jameclaw/pkg/fileutil"
)

type SkillBundle struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Skills      []string `json:"skills"`
	Instruction string   `json:"instruction,omitempty"`
}

func (m *SkillManager) bundlesRoot() string { return filepath.Join(m.skillsRoot(), ".bundles") }

func (m *SkillManager) ListBundles() ([]SkillBundle, error) {
	entries, err := os.ReadDir(m.bundlesRoot())
	if os.IsNotExist(err) {
		return []SkillBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]SkillBundle, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(m.bundlesRoot(), entry.Name()))
		if readErr != nil {
			continue
		}
		var bundle SkillBundle
		if json.Unmarshal(data, &bundle) == nil && bundle.Name != "" && len(bundle.Skills) > 0 {
			result = append(result, bundle)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m *SkillManager) SaveBundle(bundle SkillBundle) error {
	bundle.Name = normalizeSkillName(bundle.Name)
	if err := validateManagedName(bundle.Name); err != nil {
		return err
	}
	if len(bundle.Skills) == 0 {
		return fmt.Errorf("bundle %q needs at least one skill", bundle.Name)
	}
	seen := map[string]bool{}
	for i, name := range bundle.Skills {
		name = normalizeSkillName(name)
		if err := validateManagedName(name); err != nil {
			return err
		}
		if m.findSkillDir(name) == "" {
			return fmt.Errorf("bundle skill %q was not found", name)
		}
		if !seen[name] {
			bundle.Skills[i] = name
			seen[name] = true
		}
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.bundlesRoot(), 0o755); err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(filepath.Join(m.bundlesRoot(), bundle.Name+".json"), data, 0o600)
}
