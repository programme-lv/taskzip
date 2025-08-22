package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/programme-lv/taskzip/assist"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/common/zips"
	"github.com/programme-lv/taskzip/external/lio/lio2023"
	"github.com/programme-lv/taskzip/external/lio/lio2024"
	"github.com/programme-lv/taskzip/taskfs"
	"github.com/spf13/cobra"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var rootCmd = &cobra.Command{
		Use:   "taskzip",
		Short: "Task .zip archive management CLI tool",
	}

	var src string
	var dst string
	var format string
	var zipOut bool
	var transformCmd = &cobra.Command{
		Use:   "transform",
		Short: "Transform task format to task-zip format",
		Run: func(cmd *cobra.Command, args []string) {
			err := transform(src, dst, format, zipOut)
			if err != nil {
				etrace.PrintDebug(err)

				// os.Exit(1)
			}
		},
	}

	transformCmd.Flags().StringVarP(&src, "src", "s", "", "Source task directory path (*)")
	transformCmd.Flags().StringVarP(&dst, "dst", "d", "", "Destination parent dir where new task will be written (*)")
	transformCmd.Flags().StringVarP(&format, "format", "f", "", "Format of the import [lio2023] (*)")
	transformCmd.Flags().BoolVar(&zipOut, "zip", false, "Write output as <ShortID>.zip into dst directory")

	transformCmd.MarkFlagRequired("src")
	transformCmd.MarkFlagRequired("dst")
	transformCmd.MarkFlagRequired("format")

	var validateCmd = &cobra.Command{
		Use:   "validate [task-path]",
		Short: "Validate a task-zip task (dir or .zip)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := validate(args[0]); err != nil {
				etrace.PrintDebug(err)
			}
		},
	}

	var assistCmd = &cobra.Command{
		Use:   "assist [task-path]",
		Short: "Assist filling out missing info using AI",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := assistFunc(args[0]); err != nil {
				etrace.PrintDebug(err)
			}
		},
	}

	rootCmd.AddCommand(transformCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(assistCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func transform(src string, dst string, format string, zipOut bool) error {
	fmt.Printf("INFO:\trunning transform\n\t- src: %s\n\t- dst: %s\n\t- format: %s\n", src, dst, format)

	srcDir, cleanup, err := prepareSrcDir(src)
	if err != nil {
		return etrace.Wrap("prepare src", err)
	}
	defer cleanup()

	var task taskfs.Task
	switch format {
	case "lio2023":
		task, err = lio2023.ParseLio2023TaskDir(srcDir)
	case "lio2024":
		task, err = lio2024.ParseLio2024TaskDir(srcDir)
	default:
		msg := fmt.Sprintf("unsupported task format: %s", format)
		return etrace.NewError(msg)
	}
	if err != nil {
		return etrace.Wrap("parsing task in transform cmd", err)
	}

	if err := task.Validate(); err != nil {
		if etrace.IsCritical(err) {
			msg := "validate task parsed"
			return etrace.Wrap(msg, err)
		}
	}

	if zipOut {
		return transformZip(task, dst)
	}
	return transformDir(task, dst)
}

func transformDir(task taskfs.Task, dst string) error {
	path := filepath.Join(dst, task.ShortID)

	if dirExists(path) {
		ok, err := promptEraseExistingDir(path)
		if err != nil {
			return etrace.Wrap("prompt erase", err)
		}
		if ok {
			if err := os.RemoveAll(path); err != nil {
				return etrace.Wrap("remove dir", err)
			}
		}
	}

	if err := taskfs.Write(task, path); err != nil {
		return etrace.Wrap("write task", err)
	}

	fmt.Println("INFO:\tsuccessfully transformed task")
	printTaskOverview(task)
	readmePath := filepath.Join(path, "readme.md")
	if fileInfo, err := os.Stat(readmePath); err == nil && fileInfo.Size() > 0 {
		fmt.Printf("HINT:\tcheck out %s\n", readmePath)
	}
	return nil
}

func transformZip(task taskfs.Task, dst string) error {
	zipPath := filepath.Join(dst, task.ShortID+".zip")
	if fileExists(zipPath) {
		ok, err := promptEraseExistingFile(zipPath)
		if err != nil {
			return etrace.Wrap("prompt erase", err)
		}
		if ok {
			if err := os.Remove(zipPath); err != nil {
				return etrace.Wrap("remove file", err)
			}
		}
	}

	if err := taskfs.WriteZip(task, zipPath); err != nil {
		return etrace.Wrap("write zip", err)
	}
	fmt.Printf("INFO:\tsuccess; zip at %s\n", zipPath)
	fmt.Println("HINT:\tcheck out readme.md inside the .zip")
	printTaskOverview(task)
	return nil
}

func promptEraseExistingDir(dirPath string) (bool, error) {
	fmt.Printf("WARN:\tdest dir %s already exists\n", dirPath)
	fmt.Print("ASK:\terase it recursively and continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func promptEraseExistingFile(filePath string) (bool, error) {
	fmt.Printf("WARN:\tdest file %s already exists\n", filePath)
	fmt.Print("ASK:\terase it and continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

// prepareSrcDir accepts either a directory path or a .zip path. In case of a zip,
// it unzips it into a temporary directory and returns the directory containing
// the task root. It supports zips that either contain the task at root or inside
// a single top-level directory. The caller must call the returned cleanup func.
func prepareSrcDir(src string) (string, func(), error) {
	// default no-op cleanup
	noop := func() {}

	abs, err := filepath.Abs(src)
	if err != nil {
		return "", noop, etrace.Wrap("abs src", err)
	}

	if strings.HasSuffix(strings.ToLower(abs), ".zip") {
		tmp, err := os.MkdirTemp("", "taskzip-src-")
		if err != nil {
			return "", noop, etrace.Wrap("mktemp", err)
		}
		cleanup := func() { _ = os.RemoveAll(tmp) }
		if err := zips.Unzip(abs, tmp); err != nil {
			return "", cleanup, etrace.Wrap("unzip", err)
		}
		// detect if there's a single top-level dir
		entries, err := os.ReadDir(tmp)
		if err != nil {
			return "", cleanup, etrace.Wrap("readdir tmp", err)
		}
		if len(entries) == 1 && entries[0].IsDir() {
			return filepath.Join(tmp, entries[0].Name()), cleanup, nil
		}
		return tmp, cleanup, nil
	}

	// assume directory
	info, err := os.Stat(abs)
	if err != nil {
		return "", noop, etrace.Wrap("stat src", err)
	}
	if !info.IsDir() {
		return "", noop, etrace.NewError("src must be a directory or .zip")
	}
	return abs, noop, nil
}

func assistFunc(src string) error {
	fmt.Printf("INFO:\trunning assist on %s\n", src)
	dir, cleanup, err := prepareSrcDir(src)
	if err != nil {
		return err
	}
	defer cleanup()

	task, err := taskfs.Read(dir)
	if err != nil {
		return etrace.Wrap("read task", err)
	}

	fmt.Println("WARN:\tsuccessful action will overwrite source")
	fmt.Println("HINT:\tpress Ctrl+C to exit")
	fmt.Println("INFO:\tavailable workflows:")
	fmt.Println("\t1. use .typ from archive to fill lv.md statement")
	fmt.Printf("ASK:\tchoose workflow: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return etrace.Wrap("read line", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	switch answer {
	case "1":
		fmt.Println("INFO:\tfilling lv.md statement")
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

	fmt.Printf("INFO:\tsuccessfully completed workflow\n")
	return nil
}

func fillLvMdStatement(task taskfs.Task) (taskfs.Task, error) {
	// find .typ files in archive
	typFiles := []assist.File{}
	for _, file := range task.Archive.Files {
		if strings.HasSuffix(file.RelPath, ".typ") {
			typFiles = append(typFiles, assist.File{
				Content: file.Content,
				Fname:   file.RelPath,
			})
		}
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
	prompt += "It should look like this with TODO replaced with actual content:\n"

	prompt = strings.ReplaceAll(prompt, "\n", "\n\n")

	example := `Stāsts
------

TODO

Ievaddati
---------

TODO

Izvaddati
---------

TODO
`

	prompt += fmt.Sprintf("```\n%s\n```\n", example)

	response, err := assist.AskChatGpt(prompt, typFiles)
	if err != nil {
		return task, etrace.Wrap("ask chat gpt", err)
	}

	fmt.Println(response)
	panic("not implemented")
}

func validate(src string) error {
	fmt.Printf("INFO:\trunning validate on %s\n", src)
	dir, cleanup, err := prepareSrcDir(src)
	if err != nil {
		return err
	}
	defer cleanup()

	task, err := taskfs.Read(dir)
	if err != nil {
		return etrace.Wrap("read task", err)
	}
	fmt.Println("INFO:\tread task without errors")
	printTaskOverview(task)
	if err := task.Validate(); err != nil {
		return err
	}
	fmt.Println("INFO:\tall good")
	return nil
}

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
	fmt.Printf("\t- origin: olymp %q, stage %q, org %q, year %s, authors %d\n", task.Origin.Olympiad, task.Origin.OlyStage, task.Origin.Org, task.Origin.Year, len(task.Origin.Authors))
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
