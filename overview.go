package main

import (
	"fmt"
	"strings"

	"github.com/programme-lv/taskzip/taskfs"
)

func printTaskOverview(task taskfs.Task) {
	defName := pickDefaultName(task.FullName)
	nameLangs := len(task.FullName)
	readme := strings.TrimSpace(task.ReadMe) != ""

	storyLangs := len(task.Statement.Stories)
	subtasks := len(task.Statement.Subtasks)
	subtaskLangs := countSubtaskLangs(task.Statement.Subtasks)
	examples := len(task.Statement.Examples)
	exampleNotes := countExampleNotes(task.Statement.Examples)
	images := len(task.Statement.Images)

	fmt.Printf("\t- id: %s\n", task.ShortID)
	fmt.Printf("\t- name: %s (%d langs)\n", defName, nameLangs)
	fmt.Printf("\t- has readme: %t\n", readme)
	fmt.Printf("\t- statement: story (%d langs), %d images\n", storyLangs, images)
	fmt.Printf("\t- statement: %d subtasks (%d langs), %d examples (%d notes)\n", subtasks, subtaskLangs, examples, exampleNotes)
	// origin overview
	noteLangs := countNonEmptyLangs(task.Origin.Notes)
	fmt.Printf("\t- origin: olymp %q, stage %q, org %q, year %s, lang %q, authors %d\n", task.Origin.Olympiad, task.Origin.OlyStage, task.Origin.Org, task.Origin.Year, task.Origin.Lang, len(task.Origin.Authors))
	if noteLangs > 0 {
		note := pickDefaultNote(task.Origin.Notes)
		note = strings.TrimSpace(note)
		note = strings.ReplaceAll(note, "\n", " ")
		fmt.Printf("\t  notes (%d langs): %s\n", noteLangs, truncate(note, 140))
	}

	fmt.Printf("\t- testing: %s, %d tests\n", task.Testing.TestingT, len(task.Testing.Tests))
	fmt.Printf("\t- scoring: %s, %dp, %d groups\n", task.Scoring.ScoringT, task.Scoring.TotalP, len(task.Scoring.Groups))
	fmt.Printf("\t- solutions: %d\n", len(task.Solutions))
	// archive overview
	ogPdfs := task.Archive.GetOgStatementPdfs()
	illustrImgs := task.Archive.GetIllustrImgs()
	fmt.Printf("\t- archive: %d files, orig pdfs: %d, illustr: %t\n", len(task.Archive.Files), len(ogPdfs), len(illustrImgs) > 0)
}

func pickDefaultName(m taskfs.I18N[string]) string {
	if s, ok := m["en"]; ok && strings.TrimSpace(s) != "" {
		return s
	}
	if s, ok := m["lv"]; ok && strings.TrimSpace(s) != "" {
		return s
	}
	for _, s := range m {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func countSubtaskLangs(subtasks []taskfs.Subtask) int {
	langs := map[string]struct{}{}
	for _, st := range subtasks {
		for lang := range st.Desc {
			langs[lang] = struct{}{}
		}
	}
	return len(langs)
}

func countExampleNotes(examples []taskfs.Example) int {
	notes := 0
	for _, ex := range examples {
		for _, note := range ex.MdNote {
			if strings.TrimSpace(note) != "" {
				notes++
			}
		}
	}
	return notes
}

func countNonEmptyLangs(m taskfs.I18N[string]) int {
	c := 0
	for _, v := range m {
		if strings.TrimSpace(v) != "" {
			c++
		}
	}
	return c
}

func pickDefaultNote(m taskfs.I18N[string]) string {
	if s, ok := m["en"]; ok && strings.TrimSpace(s) != "" {
		return s
	}
	if s, ok := m["lv"]; ok && strings.TrimSpace(s) != "" {
		return s
	}
	for _, s := range m {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
