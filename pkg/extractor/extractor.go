package extractor

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	cp "github.com/otiai10/copy"
	ldd "github.com/u-root/u-root/pkg/ldd"
)

type config struct {
	files  []string
	outDir string
}

// WithFiles sets the input files of the packer
func WithFiles(s ...string) option {
	return func(k *config) error {
		k.files = s
		return nil
	}
}

// WithOutputDir sets the output dir of the packer
func WithOutputDir(s string) option {
	return func(k *config) error {
		k.outDir = s
		return nil
	}
}

// Option is an extractor option
type option func(k *config) error

// Extract a binary and its deps into a folder
func Extract(o ...option) error {
	config := &config{}
	for _, oo := range o {
		if err := oo(config); err != nil {
			return err
		}
	}

	files, err := ldd.Ldd(config.files)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	copyFile := func(src, dest string) {
		defer wg.Done()
		if err := cp.Copy(src, dest); err != nil {
			fmt.Println("Error copying file:", err)
		}
	}

	for _, f := range files {
		p := path.Dir(f.FullName)
		destPath := filepath.Join(config.outDir, p)
		if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
			return err
		}
		wg.Add(1)
		go copyFile(f.FullName, filepath.Join(destPath, f.Name()))
	}

	for _, f := range config.files {
		p := path.Dir(f)
		destPath := filepath.Join(config.outDir, p)
		if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
			return err
		}
		wg.Add(1)
		go copyFile(f, filepath.Join(destPath, filepath.Base(f)))
	}

	wg.Wait()
	return nil
}
