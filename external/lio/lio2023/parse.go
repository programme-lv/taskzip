package lio2023

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/external/lio"
	"github.com/programme-lv/taskzip/taskfs"
)

func ParseLio2023TaskDir(dirPath string) (taskfs.Task, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("task.yaml file not found: %s", taskYamlPath)
			return taskfs.Task{}, etrace.NewError(msg)
		}
		return taskfs.Task{}, etrace.Trace(err)
	}

	taskYaml, err := ParseLio2023Yaml(taskYamlContent)
	if err != nil {
		return taskfs.Task{}, etrace.Trace(err)
	}

	task := taskfs.Task{}
	// FullName is now i18n[string], so we need to create a map
	task.FullName = map[string]string{"lv": taskYaml.Title}
	task.ShortID = taskYaml.Name

	// Set up required metadata with defaults
	task.Metadata.Difficulty = 3 // default difficulty
	task.Metadata.ProblemTags = []string{"lio2023"}

	// Set up required origin with defaults
	task.Origin.Olympiad = "LIO"
	task.Origin.OlyStage = "national"
	task.Origin.Org = "LV"
	task.Origin.Notes = map[string]string{"lv": "LIO 2023 task"}
	task.Origin.Authors = []string{"LIO 2023"}
	task.Origin.Year = "2023"

	// Set up subtasks based on subtask_points (skip first 0 entry)
	for i, points := range taskYaml.SubtaskPoints {
		if i == 0 || points == 0 {
			continue // skip examples or 0-point subtasks
		}
		task.Statement.Subtasks = append(task.Statement.Subtasks, taskfs.Subtask{
			Desc:     map[string]string{"lv": fmt.Sprintf("Subtask %d", i)},
			Points:   points,
			VisInput: false,
		})
	}

	// Set up basic testing configuration
	task.Testing.TestingT = "simple"                       // default to simple testing
	task.Testing.CpuLimMs = int(taskYaml.TimeLimit * 1000) // convert seconds to milliseconds and to int
	task.Testing.MemLimMiB = taskYaml.MemoryLimit          // assuming it's already in MiB

	// Check for interactor first, then checker (interactor takes precedence)
	interactorPath := filepath.Join(dirPath, "riki", "interactor.cpp")
	checkerPath := filepath.Join(dirPath, "riki", "checker.cpp")

	if _, err := os.Stat(interactorPath); !errors.Is(err, fs.ErrNotExist) {
		content, err := os.ReadFile(interactorPath)
		if err != nil {
			return taskfs.Task{}, etrace.Trace(err)
		}
		task.Testing.Interactor = string(content)
		task.Testing.TestingT = "interactor"
	} else if _, err := os.Stat(checkerPath); !errors.Is(err, fs.ErrNotExist) {
		content, err := os.ReadFile(checkerPath)
		if err != nil {
			return taskfs.Task{}, etrace.Trace(err)
		}
		task.Testing.Checker = string(content)
		task.Testing.TestingT = "checker"
	}

	solutionsPath := filepath.Join(dirPath, "risin")
	if _, err := os.Stat(solutionsPath); !errors.Is(err, fs.ErrNotExist) {
		// loop through all files in risin using filepath.Walk
		err = filepath.Walk(solutionsPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return etrace.Trace(err)
			}
			if info.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(solutionsPath, path)
			if err != nil {
				return etrace.Trace(err)
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return etrace.Trace(err)
			}

			task.Solutions = append(task.Solutions, taskfs.Solution{
				Fname:    filepath.Base(relativePath),
				Content:  string(content), // convert []byte to string
				Subtasks: []int{},         // empty for now, could be populated based on filename
			})

			return nil
		})

		if err != nil {
			return taskfs.Task{}, etrace.Trace(err)
		}
	}

	testZipAbsolutePath := filepath.Join(dirPath, taskYaml.TestArchive)
	tests, err := lio.ReadLioTestsFromZip(testZipAbsolutePath)
	if err != nil {
		// Check if it's a missing file error
		if errors.Is(err, os.ErrNotExist) {
			msg := fmt.Sprintf("test archive file not found: %s", taskYaml.TestArchive)
			return taskfs.Task{}, etrace.NewError(msg)
		}
		return taskfs.Task{}, etrace.Trace(err)
	}

	testGroupTestIds := make(map[int][]int)
	for _, test := range tests {
		if test.TestGroup == 0 {
			// Examples go in Statement.Examples
			task.Statement.Examples = append(task.Statement.Examples, taskfs.Example{
				Input:  string(test.Input),  // convert []byte to string
				Output: string(test.Answer), // convert []byte to string
				MdNote: map[string]string{}, // empty for now
			})
		} else {
			// Tests go in Testing.Tests
			task.Testing.Tests = append(task.Testing.Tests, taskfs.Test{
				Input:  string(test.Input),  // convert []byte to string
				Answer: string(test.Answer), // convert []byte to string
			})
			testId := len(task.Testing.Tests)

			if testGroupTestIds[test.TestGroup] == nil {
				testGroupTestIds[test.TestGroup] = make([]int, 0)
			}
			testGroupTestIds[test.TestGroup] = append(testGroupTestIds[test.TestGroup], int(testId))
		}
	}

	punktiTxtPath := filepath.Join(dirPath, "punkti.txt")
	punktiTxtContent, err := os.ReadFile(punktiTxtPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return taskfs.Task{}, etrace.NewError("punkti.txt file not found")
		}
		return taskfs.Task{}, etrace.Trace(err)
	}
	// split by "\n"
	parts := strings.Split(string(punktiTxtContent), "\n")
	for _, line := range parts {
		if line == "" {
			continue
		}
		// split by space
		parts := strings.Split(line, " ")
		testInterval := strings.Split(parts[0], "-")

		if len(testInterval) != 2 {
			msg := fmt.Sprintf("invalid test interval format: %s", line)
			return taskfs.Task{}, etrace.NewError(msg)
		}

		start, err := strconv.Atoi(testInterval[0])
		if err != nil {
			msg := fmt.Sprintf("invalid start number in test interval: %s", testInterval[0])
			return taskfs.Task{}, etrace.NewError(msg)
		}
		end, err := strconv.Atoi(testInterval[1])
		if err != nil {
			msg := fmt.Sprintf("invalid end number in test interval: %s", testInterval[1])
			return taskfs.Task{}, etrace.NewError(msg)
		}

		points, err := strconv.Atoi(parts[1])
		if err != nil {
			msg := fmt.Sprintf("invalid points value: %s", parts[1])
			return taskfs.Task{}, etrace.NewError(msg)
		}

		for i := start; i <= end; i++ {
			if i == 0 {
				continue // example test group
			}
			// Map test groups to subtasks based on a simple distribution
			// For LIO tasks, typically test groups map to subtasks sequentially
			subtaskNum := ((len(task.Scoring.Groups) % len(task.Statement.Subtasks)) + 1)
			if len(task.Statement.Subtasks) == 0 {
				subtaskNum = 1 // fallback
			}

			// TestGroups are now in Scoring.Groups and have different structure
			task.Scoring.Groups = append(task.Scoring.Groups, taskfs.TestGroup{
				Points:  points,
				Range:   [2]int{start, end}, // [from, to] inclusive
				Public:  false,              // assume private by default
				Subtask: subtaskNum,
			})
		}
	}

	// Set up scoring configuration
	task.Scoring.ScoringT = "min-groups"
	totalPoints := 0
	for _, group := range task.Scoring.Groups {
		totalPoints += group.Points
	}
	task.Scoring.TotalP = totalPoints

	excludePrefixFromArchive := []string{
		punktiTxtPath,
		testZipAbsolutePath,
		solutionsPath,
		taskYamlPath,
		checkerPath,
		interactorPath,
	}

	// Archive files go in Archive.Files
	err = filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return etrace.Trace(err)
		}
		if info.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return etrace.Trace(err)
		}
		relativePath = "./" + relativePath
		for _, prefix := range excludePrefixFromArchive {
			prefixAbs, err := filepath.Abs(prefix)
			if err != nil {
				return etrace.Trace(err)
			}
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				return etrace.Trace(err)
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
			return etrace.Trace(err)
		}
		// Archive structure changed - it's now Archive.Files
		task.Archive.Files = append(task.Archive.Files, taskfs.ArchiveFile{
			RelPath: relativePath, // field name changed from RelativePath to RelPath
			Content: content,
		})
		return nil
	})
	if err != nil {
		return taskfs.Task{}, etrace.Trace(err)
	}

	// Origin structure changed
	task.Origin.Olympiad = "LIO"
	task.Origin.OlyStage = "national" // assume national level
	task.Origin.Year = "2023"         // since this is lio2023

	return task, nil
}
