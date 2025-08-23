package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/programme-lv/taskzip/assist"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

func assistFunc(src string) error {
	info("running assist on %s", src)
	dir, cleanup, err := ensureSrcIsDir(src)
	if err != nil {
		return err
	}
	defer cleanup()

	task, err := taskfs.Read(dir)
	if err != nil {
		return etrace.Wrap("read task", err)
	}

	warn("successful action will overwrite source; press Ctrl+C to exit")
	workflows := []string{
		"use .typ from archive to fill lv.md statement",
	}
	info("available workflows:")
	for i, workflow := range workflows {
		fmt.Printf("\t%d. %s\n", i+1, workflow)
	}
	ask("choose workflow")

	answer, err := readAnswer()
	if err != nil {
		return etrace.Wrap("read answer", err)
	}
	switch answer {
	case "1":
		info("received answer: %s (filling lv.md statement)", answer)
		task, err = fillLvMdStatement(task)
		if err != nil {
			return etrace.Wrap("fill lv.md statement", err)
		}
	default:
		return etrace.NewError("invalid workflow")
	}

	if strings.HasSuffix(src, ".zip") {
		// delete the .zip
		if err := os.Remove(src); err != nil {
			return etrace.Wrap("remove zip", err)
		}
		err = taskfs.WriteZip(task, src)
		if err != nil {
			return etrace.Wrap("write zip", err)
		}
	} else {
		// delete the original dir
		if err := os.RemoveAll(dir); err != nil {
			return etrace.Wrap("remove dir", err)
		}
		// write task to that location
		if err := taskfs.Write(task, dir); err != nil {
			return etrace.Wrap("write task", err)
		}
	}

	info("successfully completed workflow")
	return nil
}

func fillLvMdStatement(task taskfs.Task) (taskfs.Task, error) {
	// find .typ files in archive
	files := []assist.File{}
	for _, file := range task.Archive.Files {
		if strings.HasSuffix(file.RelPath, ".typ") {
			files = append(files, assist.File{
				Content: file.Content,
				Fname:   file.RelPath,
			})
		}
	}
	for _, file := range task.Archive.GetOgStatementPdfs() {
		files = append(files, assist.File{
			Content: file.Content,
			Fname:   fmt.Sprintf("%s.pdf", file.Language),
		})
	}
	prompt := "You are a precise technical writer. Use the attached files. " +
		"Return your final answer as RAW GitHub Flavored Markdown ONLY. " +
		"Do NOT wrap in code fences. Do NOT include any prose before or after the markdown.\n"

	prompt += "Your task is to transfer an competitive programming task statement" +
		"from Typst (.typ) to Markdown (.md) format. Language of statement is Latvian. " +
		"Note that the added file may not have a .typ extension but a .txt.\n"

	prompt += "The resulting markdown may contain mathematical expressions. " +
		"Convert the math expressions to KaTeX-compatible format using dollar signs (`$...$`).\n"

	prompt += "Result should contain 3 sections: stāsts, ievaddati, izvaddati. "
	prompt += "It should look like this with ... replaced with actual content:\n"

	prompt = strings.ReplaceAll(prompt, "\n", "\n\n")

	example := `Stāsts
------

...

Ievaddati
---------

...

Izvaddati
---------

...
`

	prompt += fmt.Sprintf("```\n%s\n```\n", example)

	response, err := assist.AskChatGpt(prompt, files)
	if err != nil {
		return task, etrace.Wrap("ask chat gpt", err)
	}

	story, err := taskfs.ParseMdStory(response, "lv")
	if err != nil {
		return task, etrace.Wrap("parse md story", err)
	}
	task.Statement.Stories["lv"] = story
	return task, nil
}
