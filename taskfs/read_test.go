package taskfs_test

import (
	"strings"
	"testing"

	"github.com/programme-lv/task-zip/taskfs"
	"github.com/stretchr/testify/require"
)

func getTestTask(t *testing.T) *taskfs.Task {
	task, err := taskfs.Read("testdata/kvadrputekl")
	require.NoError(t, err)
	return &task
}

func TestReadBasicInfo(t *testing.T) {
	task := getTestTask(t)
	BasicInfo(t, task)
}

func BasicInfo(t *testing.T, task *taskfs.Task) {
	require.Equal(t, "kvadrputekl", task.ShortID)
	require.Equal(t, "Kvadrātveida putekļsūcējs", task.FullName["lv"])
	require.Equal(t, "Square vacuum cleaner", task.FullName["en"])
	require.Equal(t, 2, len(task.FullName))
	require.Contains(t, task.ReadMe, "this is an example readme.md file")
}

func TestReadOrigin(t *testing.T) {
	task := getTestTask(t)
	Origin(t, task)
}

func Origin(t *testing.T, task *taskfs.Task) {
	require.Equal(t, "LIO", task.Origin.Olympiad)
	require.Equal(t, "2023/2024", task.Origin.Year)
	require.Equal(t, "school", task.Origin.OlyStage)
	require.Equal(t, "PPS", task.Origin.Org)
	require.Equal(t, []string{"Krišjānis Petručeņa"}, task.Origin.Authors)
	require.Contains(t, task.Origin.Notes["lv"], "Uzdevums no Latvijas 37")
	require.Contains(t, task.Origin.Notes["en"], "The problem is from")
	require.Equal(t, 2, len(task.Origin.Notes))
}

func TestReadMetadata(t *testing.T) {
	task := getTestTask(t)
	Metadata(t, task)
}

func Metadata(t *testing.T, task *taskfs.Task) {
	require.Equal(t, []string{"bfs", "grid", "prefix-sum", "sliding-window", "shortest-path", "graphs"}, task.Metadata.ProblemTags)
	require.Equal(t, 3, task.Metadata.Difficulty)
}

func TestReadSolutions(t *testing.T) {
	task := getTestTask(t)
	Solutions(t, task)
}

func Solutions(t *testing.T, task *taskfs.Task) {
	require.Equal(t, 3, len(task.Solutions))

	sol1 := task.Solutions[0]
	require.Equal(t, "kp_kp_ok.cpp", sol1.Fname)
	require.Equal(t, []int{1, 2, 3}, sol1.Subtasks)
	require.Contains(t, sol1.Content, "#include <iostream>")

	sol2 := task.Solutions[1]
	require.Equal(t, "kp_kp_tle.cpp", sol2.Fname)
	require.Equal(t, []int{1, 2}, sol2.Subtasks)
	require.Contains(t, sol2.Content, "#include <bits/stdc++.h>")
}

func TestReadTesting(t *testing.T) {
	task := getTestTask(t)
	Testing(t, task)
}

func Testing(t *testing.T, task *taskfs.Task) {
	require.Equal(t, "checker", task.Testing.TestingT)
	require.Equal(t, 500, task.Testing.CpuLimMs)
	require.Equal(t, 256, task.Testing.MemLimMiB)

	require.NotEmpty(t, task.Testing.Checker)
	require.Contains(t, task.Testing.Checker, "#include \"iostream\"")
	require.Contains(t, task.Testing.Checker, "hello world")
	require.Empty(t, task.Testing.Interactor)

	require.Equal(t, 2, len(task.Testing.Tests))

	for i, test := range task.Testing.Tests {
		require.NotEmpty(t, test.Input, "Test %d input should not be empty", i+1)
		require.NotEmpty(t, test.Answer, "Test %d answer should not be empty", i+1)
	}

	test1 := task.Testing.Tests[0]
	test2 := task.Testing.Tests[1]

	require.Contains(t, test1.Input, "5 5 3")
	require.Contains(t, test2.Input, "5 5 3")
	require.Equal(t, "a", strings.TrimSpace(test1.Answer))
	require.Equal(t, "a", strings.TrimSpace(test2.Answer))

	// TODO: archive
}

func TestReadStatement(t *testing.T) {
	task := getTestTask(t)
	Statement(t, task)
}

func Statement(t *testing.T, task *taskfs.Task) {
	require.Len(t, task.Statement.Subtasks, 2)
	subtask := task.Statement.Subtasks[0]
	require.Equal(t, 6, subtask.Points)
	require.Equal(t, "Uzdevuma tekstā dotie trīs piemēri.", subtask.Desc["lv"])
	require.Equal(t, "The three examples given in the problem statement.", subtask.Desc["en"])
	subtask2 := task.Statement.Subtasks[1]
	require.Equal(t, 94, subtask2.Points)
	require.Equal(t, "Uzdevuma tekstā dotie trīs piemēri.", subtask2.Desc["lv"])
	require.Equal(t, "The three examples given in the problem statement.", subtask2.Desc["en"])

	require.Len(t, task.Statement.Examples, 2)
	example := task.Statement.Examples[0]
	require.Equal(t, "5 9 3", example.Input)
	require.Equal(t, "10", example.Output)
	require.Equal(t, "Šis ir paskaidrojums piemēram.", example.MdNote["lv"])
	require.Equal(t, "This is an explanation of the example.", example.MdNote["en"])
	require.Len(t, example.MdNote, 2)

	require.Len(t, task.Statement.Stories, 1)
	story := task.Statement.Stories["lv"]
	require.Equal(t, "![1. attēls: Laukuma piemērs](kp1.png)", story.Story)
	require.Equal(t, "Ievaddatu piemērs", story.Input)
	require.Equal(t, "Izvaddatu piemērs", story.Output)
	require.Equal(t, "Šis ir interaktīvs uzdevums.", story.Talk)

	require.Len(t, task.Statement.Images, 2)
	require.Equal(t, task.Statement.Images[0].Fname, "kp1.png")
	require.NotEmpty(t, task.Statement.Images[0].Content)
	require.Equal(t, task.Statement.Images[1].Fname, "kp2.png")
	require.NotEmpty(t, task.Statement.Images[1].Content)
}

func TestReadScoring(t *testing.T) {
	task := getTestTask(t)
	Scoring(t, task)
}

func Scoring(t *testing.T, task *taskfs.Task) {
	require.Equal(t, "min-groups", task.Scoring.ScoringT)
	require.Equal(t, 100, task.Scoring.TotalP)

	require.Len(t, task.Scoring.Groups, 3)
	second := task.Scoring.Groups[1]
	require.Equal(t, [2]int{6, 10}, second.Range)
	require.Equal(t, 3, second.Points)
	require.Equal(t, 1, second.Subtask)
	require.True(t, second.Public)

	third := task.Scoring.Groups[2]
	require.Equal(t, [2]int{11, 13}, third.Range)
	require.Equal(t, 94, third.Points)
	require.Equal(t, 2, third.Subtask)
	require.False(t, third.Public)
}

func TestReadArchive(t *testing.T) {
	task := getTestTask(t)
	Archive(t, task)
}

func Archive(t *testing.T, task *taskfs.Task) {
	expFilePaths := []string{
		"task.yaml",
		"riki/00_gen_params.py",
		"riki/hello.txt",
		"reserved/illustration/img.jpg",
		"reserved/statement/lv.pdf",
	}

	taskYamlExpContent := "name: 'Kp'"

	actualPaths := []string{}
	actualTaskYamlContent := ""
	for _, f := range task.Archive.Files {
		actualPaths = append(actualPaths, f.RelPath)
		if f.RelPath == "task.yaml" {
			actualTaskYamlContent = string(f.Content)
		}
	}

	require.ElementsMatch(t, expFilePaths, actualPaths)
	require.Contains(t, actualTaskYamlContent, taskYamlExpContent)

	ogPdfs := task.Archive.GetOgStatementPdfs()
	require.Len(t, ogPdfs, 1)
	require.Equal(t, "lv", ogPdfs[0].Language)

	illustrImgs := task.Archive.GetIllustrImgs()
	require.Len(t, illustrImgs, 1)
	require.Equal(t, "img.jpg", illustrImgs[0].Fname)
}
