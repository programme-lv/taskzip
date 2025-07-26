package taskfs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	_ "github.com/pelletier/go-toml/v2"
)

func Read(dirPath string) (Task, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return Task{}, err
	}
	defer dir.Close()

	tomlPath := filepath.Join(dirPath, "task.toml")
	tomlReader, err := os.Open(tomlPath)
	if err != nil {
		msg := "failed to open task.toml"
		return Task{}, errf(msg, err)
	}
	defer tomlReader.Close()

	taskToml := TaskToml{}
	d := toml.NewDecoder(tomlReader)
	d.DisallowUnknownFields()
	err = d.Decode(&taskToml)
	if err != nil {
		msg := "failed to decode task.toml"
		return Task{}, errf(msg, err)
	}

	testing := Testing{
		TestingT:   taskToml.Testing.Type,
		MemLimMiB:  taskToml.Testing.MemMiB,
		CpuLimMs:   taskToml.Testing.CpuMs,
		Tests:      []Test{},
		Checker:    "",
		Interactor: "",
	}
	testing.Tests, err = ReadTests(filepath.Join(dirPath, "tests"))
	if err != nil {
		msg := "failed to read tests"
		return Task{}, errf(msg, err)
	}
	err = testing.Validate()
	if err != nil {
		msg := "invalid testing configuration"
		return Task{}, errf(msg, err)
	}

	return Task{}, nil
}

func errf(msg string, err error) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func ReadTests(testDirPath string) ([]Test, error) {
	// TODO: implement
	return nil, nil
}
