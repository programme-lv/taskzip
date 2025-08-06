package lio2024

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/external/lio"
	"github.com/programme-lv/task-zip/taskfs"
)

func ParseLio2024TaskDir(dirPath string) (taskfs.Task, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("task.yaml file not found: %s", taskYamlPath)
			return taskfs.Task{}, errwrap.Error(msg)
		}
		return taskfs.Task{}, errwrap.AddTrace(err)
	}

	parsedYaml, err := ParseLio2024Yaml(taskYamlContent)
	if err != nil {
		return taskfs.Task{}, errwrap.AddTrace(err)
	}

	task := taskfs.Task{}
	// FullName is now i18n[string], so we need to create a map
	task.FullName = map[string]string{"lv": parsedYaml.FullTaskName}
	task.ShortID = strings.ToLower(parsedYaml.TaskShortIDCode)

	// Set up required metadata with defaults
	task.Metadata.Difficulty = 0
	task.Metadata.ProblemTags = []string{}

	// Set up required origin with defaults
	task.Origin.Olympiad = "LIO"
	task.Origin.OlyStage = "abracadabra"
	task.Origin.Notes = map[string]string{"lv": ""}
	task.Origin.Authors = []string{}
	task.Origin.Year = ""

	// Set up basic testing configuration
	task.Testing.TestingT = "simple"                                     // default to simple testing
	task.Testing.CpuLimMs = int(parsedYaml.CpuTimeLimitInSeconds * 1000) // convert seconds to milliseconds
	task.Testing.MemLimMiB = parsedYaml.MemoryLimitInMegabytes

	if parsedYaml.CheckerPathRelToYaml != nil {
		relativePath := *parsedYaml.CheckerPathRelToYaml
		checkerPath := filepath.Join(dirPath, relativePath)
		checkerBytes, err := os.ReadFile(checkerPath)
		if err != nil {
			return taskfs.Task{}, errwrap.AddTrace(err)
		}
		task.Testing.Checker = string(checkerBytes)
		task.Testing.TestingT = "checker"
	}

	if parsedYaml.InteractorPathRelToYaml != nil {
		relativePath := *parsedYaml.InteractorPathRelToYaml
		interactorPath := filepath.Join(dirPath, relativePath)
		interactorBytes, err := os.ReadFile(interactorPath)
		if err != nil {
			return taskfs.Task{}, errwrap.AddTrace(err)
		}
		task.Testing.Interactor = string(interactorBytes)
		task.Testing.TestingT = "interactor"
	}

	testZipRelPath := parsedYaml.TestZipPathRelToYaml
	testZipAbsPath := filepath.Join(dirPath, testZipRelPath)
	tests, err := lio.ReadLioTestsFromZip(testZipAbsPath)
	if err != nil {
		// Check if it's a missing file error
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("test archive file not found: %s", testZipRelPath)
			return taskfs.Task{}, errwrap.Error(msg)
		}
		return taskfs.Task{}, errwrap.AddTrace(err)
	}

	sort.Slice(tests, func(i, j int) bool {
		if tests[i].TestGroup == tests[j].TestGroup {
			return tests[i].NoInTestGroup < tests[j].NoInTestGroup
		}
		return tests[i].TestGroup < tests[j].TestGroup
	})

	mapTestsToTestGroups := map[int][]int{}

	for _, t := range tests {
		if t.TestGroup == 0 {
			// Examples go in Statement.Examples
			task.Statement.Examples = append(task.Statement.Examples, taskfs.Example{
				Input:  string(t.Input),     // convert []byte to string
				Output: string(t.Answer),    // convert []byte to string
				MdNote: map[string]string{}, // empty for now
			})
			continue
		}
		// Tests go in Testing.Tests
		task.Testing.Tests = append(task.Testing.Tests, taskfs.Test{
			Input:  string(t.Input),  // convert []byte to string
			Answer: string(t.Answer), // convert []byte to string
		})
		id := len(task.Testing.Tests)
		mapTestsToTestGroups[t.TestGroup] = append(mapTestsToTestGroups[t.TestGroup], id)
	}

	// Set up subtasks based on parsedYaml.SubtaskPoints (skip first 0 entry)
	for i, points := range parsedYaml.SubtaskPoints {
		if i == 0 || points == 0 {
			continue // skip examples or 0-point subtasks
		}
		task.Statement.Subtasks = append(task.Statement.Subtasks, taskfs.Subtask{
			Desc:     map[string]string{"lv": fmt.Sprintf("Subtask %d", i)},
			Points:   points,
			VisInput: false,
		})
	}

	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		// TestGroups are now in Scoring.Groups and have different structure
		task.Scoring.Groups = append(task.Scoring.Groups, taskfs.TestGroup{
			Points:  g.Points,
			Range:   [2]int{g.GroupID, g.GroupID}, // single group range
			Public:  g.Public,
			Subtask: g.Subtask,
		})
	}

	// Set up scoring configuration
	task.Scoring.ScoringT = "min-groups"
	totalPoints := 0
	for _, group := range task.Scoring.Groups {
		totalPoints += group.Points
	}
	task.Scoring.TotalP = totalPoints

	// verify that subtask points sum up to 100
	subtaskTotalPoints := 0
	for _, subtask := range task.Statement.Subtasks {
		subtaskTotalPoints += subtask.Points
	}
	if subtaskTotalPoints != 100 {
		msg := fmt.Sprintf("subtask points do not sum up to 100: got %d", subtaskTotalPoints)
		return taskfs.Task{}, errwrap.Error(msg)
	}

	// The PDF statement will be included in the archive files later
	// No need to handle it separately as it's part of archive

	solutionsDirPath := filepath.Join(dirPath, "risin")
	if _, err := os.Stat(solutionsDirPath); !errors.Is(err, fs.ErrNotExist) {
		err = filepath.Walk(solutionsDirPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return errwrap.AddTrace(err)
			}
			if info.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(solutionsDirPath, path)
			if err != nil {
				return errwrap.AddTrace(err)
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return errwrap.AddTrace(err)
			}

			task.Solutions = append(task.Solutions, taskfs.Solution{
				Fname:    filepath.Base(relativePath),
				Content:  string(content), // convert []byte to string
				Subtasks: []int{},         // empty for now, could be populated based on filename
			})

			return nil
		})

		if err != nil {
			return taskfs.Task{}, errwrap.AddTrace(err)
		}
	}

	excludePrefixFromArchive := []string{
		testZipAbsPath,
		solutionsDirPath,
		filepath.Join(dirPath, "task.yaml"),
	}

	// Archive files go in Archive.Files
	err = filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errwrap.AddTrace(err)
		}
		if info.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		relativePath = "./" + relativePath
		for _, prefix := range excludePrefixFromArchive {
			prefixAbs, err := filepath.Abs(prefix)
			if err != nil {
				return errwrap.AddTrace(err)
			}
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				return errwrap.AddTrace(err)
			}
			if pathAbs == prefixAbs {
				return nil
			}
			if strings.HasPrefix(pathAbs, prefixAbs+string(filepath.Separator)) {
				return nil
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return errwrap.AddTrace(err)
		}
		// Archive structure changed - it's now Archive.Files
		task.Archive.Files = append(task.Archive.Files, taskfs.ArchiveFile{
			RelPath: relativePath, // field name changed from RelativePath to RelPath
			Content: content,
		})
		return nil
	})
	if err != nil {
		return taskfs.Task{}, errwrap.AddTrace(err)
	}

	return task, nil
}
