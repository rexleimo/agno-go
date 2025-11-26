package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fixture struct {
	FixtureID string `json:"fixtureId" yaml:"fixtureId"`
	Provider  string `json:"provider" yaml:"provider"`
	Type      string `json:"type" yaml:"type"`
	Tolerance struct {
		TokenTolerance  int     `json:"tokenTolerance" yaml:"tokenTolerance"`
		EmbeddingCosine float64 `json:"embeddingCosine" yaml:"embeddingCosine"`
	} `json:"tolerance" yaml:"tolerance"`
}

func TestFixturesLoadOrSkip(t *testing.T) {
	dir := fixtureDir()
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Skipf("fixtures directory not found: %s", dir)
	}
	if err != nil {
		t.Fatalf("stat fixtures dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("fixtures path is not a directory: %s", dir)
	}

	files, err := collectFixtureFiles(dir)
	if err != nil {
		t.Fatalf("collect fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Skipf("no contract fixtures present in %s", dir)
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			fx, err := loadFixture(path)
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}
			if fx.Tolerance.TokenTolerance < 0 {
				t.Fatalf("negative token tolerance in %s", path)
			}
			if fx.Tolerance.EmbeddingCosine > 1 || fx.Tolerance.EmbeddingCosine < 0 {
				t.Fatalf("invalid embedding tolerance in %s", path)
			}
		})
	}
}

func fixtureDir() string {
	if env := os.Getenv("FIXTURE_DEST_DIR"); env != "" {
		return env
	}
	return filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "contracts", "fixtures")
}

func collectFixtureFiles(dir string) ([]string, error) {
	var files []string
	patterns := []string{"*.json", "*.yaml", "*.yml"}
	for _, p := range patterns {
		globbed, err := filepath.Glob(filepath.Join(dir, p))
		if err != nil {
			return nil, err
		}
		files = append(files, globbed...)
	}
	return files, nil
}

func loadFixture(path string) (*fixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fx fixture
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(raw, &fx)
	default:
		err = json.Unmarshal(raw, &fx)
	}
	if err != nil {
		return nil, err
	}
	return &fx, nil
}
