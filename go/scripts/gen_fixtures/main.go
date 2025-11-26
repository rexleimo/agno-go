package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type copier struct {
	source     string
	dest       string
	copied     int
	verified   int
	verifyOnly bool
}

func main() {
	sourceFlag := flag.String("source", "", "path to precomputed fixture source (json/yaml)")
	destFlag := flag.String("dest", "", "destination fixture directory")
	verifyOnly := flag.Bool("verify-only", false, "only verify that destination fixtures match the source")
	flag.Parse()

	c := copier{
		source:     choosePath(*sourceFlag, "FIXTURE_SOURCE_DIR", "../specs/001-agno-agents-refactor/contracts/fixtures-src"),
		dest:       choosePath(*destFlag, "FIXTURE_DEST_DIR", "../specs/001-agno-agents-refactor/contracts/fixtures"),
		verifyOnly: *verifyOnly,
	}

	if err := c.run(); err != nil {
		log.Fatalf("fixture generation failed: %v", err)
	}
	log.Printf("fixtures verified: %d files (copied %d) -> %s", c.verified, c.copied, c.dest)
}

func choosePath(flagVal, envKey, defaultPath string) string {
	val := strings.TrimSpace(flagVal)
	if val != "" {
		return val
	}
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		return env
	}
	return defaultPath
}

func (c *copier) run() error {
	info, err := os.Stat(c.source)
	if err != nil {
		return fmt.Errorf("fixture source not found: %s (set FIXTURE_SOURCE_DIR or pass --source)", c.source)
	}
	if !info.IsDir() {
		return fmt.Errorf("fixture source must be a directory: %s", c.source)
	}
	if err := os.MkdirAll(c.dest, 0o755); err != nil {
		return fmt.Errorf("create dest: %w", err)
	}

	err = filepath.Walk(c.source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !isFixtureFile(info.Name()) {
			return nil
		}
		rel, err := filepath.Rel(c.source, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(c.dest, rel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		if !c.verifyOnly {
			if err := copyFile(path, destPath); err != nil {
				return err
			}
			c.copied++
		}
		if err := verifyMatch(path, destPath); err != nil {
			return err
		}
		c.verified++
		return nil
	})
	if err != nil {
		return err
	}
	if c.verified == 0 {
		return fmt.Errorf("no fixture files found in %s (expecting .json/.yaml/.yml)", c.source)
	}
	return nil
}

func isFixtureFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".json" || ext == ".yaml" || ext == ".yml"
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func verifyMatch(src, dest string) error {
	srcHash, err := fileHash(src)
	if err != nil {
		return fmt.Errorf("hash source %s: %w", src, err)
	}
	destHash, err := fileHash(dest)
	if err != nil {
		return fmt.Errorf("hash dest %s: %w", dest, err)
	}
	if srcHash != destHash {
		return fmt.Errorf("mismatched content: %s -> %s", src, dest)
	}
	return nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
