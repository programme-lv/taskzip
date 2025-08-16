package taskfs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/programme-lv/task-zip/common/etrace"
)

var (
	ErrDstDirExists = etrace.NewError("destination dir must not exist")
)

// We expect dirPath to not exist whereas its parent dir does.
func Write(task Task, dirPath string) error {
	dirAbsPath, err := filepath.Abs(dirPath)
	if err != nil {
		msg := fmt.Sprintf("get abs path of %s", dirPath)
		return etrace.Wrap(msg, err)
	}

	if doesDirExist(dirAbsPath) {
		cause := fmt.Errorf("dir %s already exists", dirAbsPath)
		return etrace.Trace(ErrDstDirExists.WithCause(cause))
	}

	parentDir := filepath.Dir(dirAbsPath)
	if !doesDirExist(parentDir) {
		msg := fmt.Sprintf("parent dir %s does not exist", parentDir)
		return etrace.Wrap(msg, nil)
	}

	err = os.Mkdir(dirAbsPath, 0755)
	if err != nil {
		msg := fmt.Sprintf("create dir %s", dirAbsPath)
		return etrace.Wrap(msg, err)
	}

	writer, err := NewTaskWriter(dirAbsPath, task)
	if err != nil {
		err2 := os.Remove(dirAbsPath)
		if err2 != nil {
			msg := fmt.Sprintf("remove dir %s", dirAbsPath)
			return etrace.Wrap(msg, err2)
		}
		msg := fmt.Sprintf("init writer to %s for task %s", dirAbsPath, task.ShortID)
		return etrace.Wrap(msg, err)
	}

	err = writer.WriteTask()
	if err != nil {
		msg := fmt.Sprintf("write task to %s", dirAbsPath)
		return etrace.Wrap(msg, err)
	}

	return nil
}

func doesDirExist(dirPath string) bool {
	_, err := os.Stat(dirPath)
	return err == nil
}

type TaskWriter struct {
	path string // absolute path to the task directory
	task *Task
}

func NewTaskWriter(
	taskDirAbsPath string,
	taskToWrite Task,
) (TaskWriter, error) {
	// check that the dir is empty
	files, err := os.ReadDir(taskDirAbsPath)
	if err != nil {
		msg := fmt.Sprintf("read task dir %s", taskDirAbsPath)
		return TaskWriter{}, etrace.Wrap(msg, err)
	}
	if len(files) > 0 {
		msg := fmt.Sprintf("dir %s is not empty", taskDirAbsPath)
		return TaskWriter{}, etrace.Wrap(msg, nil)
	}

	return TaskWriter{
		path: taskDirAbsPath,
		task: &taskToWrite,
	}, nil
}

func (w TaskWriter) WriteTask() error {
	err := w.TaskToml()
	if err != nil {
		return err
	}
	err = w.Testlib()
	if err != nil {
		return err
	}
	err = w.Tests()
	if err != nil {
		return err
	}
	err = w.Solutions()
	if err != nil {
		return err
	}
	err = w.Readme()
	if err != nil {
		return err
	}
	err = w.Examples()
	if err != nil {
		return err
	}
	err = w.TestGroups()
	if err != nil {
		return err
	}
	err = w.Statement()
	if err != nil {
		return err
	}
	err = w.Archive()
	if err != nil {
		return err
	}
	return nil
}

func (w TaskWriter) Readme() error {
	err := w.WriteFile("readme.md", []byte(w.task.ReadMe))
	if err != nil {
		return etrace.Trace(err)
	}
	return nil
}

func (w TaskWriter) WriteFile(path string, content []byte) error {
	absPath := filepath.Join(w.path, path)
	err := os.WriteFile(absPath, content, 0755)
	if err != nil {
		return etrace.Trace(err)
	}
	return nil
}

func (w TaskWriter) CreateDir(path string) error {
	absPath := filepath.Join(w.path, path)
	err := os.Mkdir(absPath, 0755)
	if err != nil {
		return etrace.Trace(err)
	}
	return nil
}

func (w TaskWriter) TaskToml() error {
	tomlPath := "task.toml"
	taskToml := NewTaskToml(w.task)

	buf := bytes.NewBuffer(nil)
	enc := toml.NewEncoder(buf)
	enc.SetTablesInline(false)
	enc.SetIndentTables(true)

	err := enc.Encode(taskToml)
	if err != nil {
		return etrace.Trace(err)
	}
	err = w.WriteFile(tomlPath, buf.Bytes())
	if err != nil {
		return etrace.Trace(err)
	}
	return nil
}

func (w TaskWriter) Testlib() error {
	if w.task.Testing.Checker != "" {
		err := w.WriteFile("checker.cpp", []byte(w.task.Testing.Checker))
		if err != nil {
			return etrace.Trace(err)
		}
	}

	if w.task.Testing.Interactor != "" {
		err := w.WriteFile("interactor.cpp", []byte(w.task.Testing.Interactor))
		if err != nil {
			return etrace.Trace(err)
		}
	}
	return nil
}

func (w TaskWriter) Tests() error {
	err := w.CreateDir("tests")
	if err != nil {
		return etrace.Trace(err)
	}

	for i, test := range w.task.Testing.Tests {
		inPath := fmt.Sprintf("tests/%03di.txt", i+1)
		outPath := fmt.Sprintf("tests/%03do.txt", i+1)
		err := w.WriteFile(inPath, []byte(test.Input))
		if err != nil {
			return etrace.Trace(err)
		}
		err = w.WriteFile(outPath, []byte(test.Answer))
		if err != nil {
			return etrace.Trace(err)
		}
	}

	return nil
}

func (w TaskWriter) Solutions() error {
	solutionsDir := "solutions"
	err := w.CreateDir("solutions")
	if err != nil {
		return etrace.Trace(err)
	}

	for _, sol := range w.task.Solutions {
		solPath := filepath.Join(solutionsDir, sol.Fname)
		err := w.WriteFile(solPath, []byte(sol.Content))
		if err != nil {
			return etrace.Trace(err)
		}
	}
	return nil
}

func (w TaskWriter) Examples() error {
	examplesDir := "examples"
	err := w.CreateDir("examples")
	if err != nil {
		msg := "create examples directory"
		return etrace.Wrap(msg, err)
	}

	for i, example := range w.task.Statement.Examples {
		inPath := filepath.Join(examplesDir, fmt.Sprintf("%03di.txt", i+1))
		outPath := filepath.Join(examplesDir, fmt.Sprintf("%03do.txt", i+1))
		notePath := filepath.Join(examplesDir, fmt.Sprintf("%03d.md", i+1))
		err := w.WriteFile(inPath, []byte(example.Input))
		if err != nil {
			msg := fmt.Sprintf("write example %d input", i)
			return etrace.Wrap(msg, err)
		}
		err = w.WriteFile(outPath, []byte(example.Output))
		if err != nil {
			msg := fmt.Sprintf("write example %d output", i)
			return etrace.Wrap(msg, err)
		}
		mdNoteContent := ""
		for lang, note := range example.MdNote {
			mdNoteContent += fmt.Sprintf("%s\n---\n%s\n", lang, note)
		}
		if mdNoteContent != "" {
			err = w.WriteFile(notePath, []byte(mdNoteContent))
			if err != nil {
				msg := fmt.Sprintf("write example %d note", i)
				return etrace.Wrap(msg, err)
			}
		}
	}
	return nil
}

func (w TaskWriter) TestGroups() error {
	filePath := "testgroups.txt"
	content := ""

	for i, group := range w.task.Scoring.Groups {
		groupId := i + 1
		from := group.Range[0]
		to := group.Range[1]
		points := group.Points
		subtask := group.Subtask

		line := fmt.Sprintf("%02d: %03d-%03d %dp (%d)", groupId, from, to, points, subtask)
		if group.Public {
			line += " *"
		}
		content += line + "\n"
	}

	err := w.WriteFile(filePath, []byte(content))
	if err != nil {
		msg := "write testgroups.txt"
		return etrace.Wrap(msg, err)
	}
	return nil
}

func (w TaskWriter) Statement() error {
	err := w.CreateDir("statement")
	if err != nil {
		msg := "create statement directory"
		return etrace.Wrap(msg, err)
	}

	// Write story files for each language
	for lang, story := range w.task.Statement.Stories {
		content := w.formatStoryMd(story, lang)
		storyPath := filepath.Join("statement", lang+".md")
		err := w.WriteFile(storyPath, []byte(content))
		if err != nil {
			msg := fmt.Sprintf("write story %s", lang)
			return etrace.Wrap(msg, err)
		}
	}

	// Write image files
	for _, image := range w.task.Statement.Images {
		imagePath := filepath.Join("statement", image.Fname)
		err := w.WriteFile(imagePath, image.Content)
		if err != nil {
			msg := fmt.Sprintf("write image %s", image.Fname)
			return etrace.Wrap(msg, err)
		}
	}

	return nil
}

func (w TaskWriter) formatStoryMd(story StoryMd, lang string) string {
	var content string

	// Get the appropriate section names for the language
	sectionNames := map[string]string{
		"Story":       "Story",
		"Input":       "Input",
		"Output":      "Output",
		"Notes":       "Notes",
		"Scoring":     "Scoring",
		"Example":     "Example",
		"Interaction": "Interaction",
	}

	if lang == "lv" {
		sectionNames = map[string]string{
			"Story":       "Stāsts",
			"Input":       "Ievaddati",
			"Output":      "Izvaddati",
			"Notes":       "Piezīmes",
			"Scoring":     "Vērtēšana",
			"Example":     "Piemērs",
			"Interaction": "Komunikācija",
		}
	}

	if story.Story != "" {
		content += sectionNames["Story"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Story"])) + "\n\n"
		content += story.Story + "\n\n"
	}

	if story.Input != "" {
		content += sectionNames["Input"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Input"])) + "\n\n"
		content += story.Input + "\n\n"
	}

	if story.Output != "" {
		content += sectionNames["Output"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Output"])) + "\n\n"
		content += story.Output + "\n\n"
	}

	if story.Notes != "" {
		content += sectionNames["Notes"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Notes"])) + "\n\n"
		content += story.Notes + "\n\n"
	}

	if story.Scoring != "" {
		content += sectionNames["Scoring"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Scoring"])) + "\n\n"
		content += story.Scoring + "\n\n"
	}

	if story.Example != "" {
		content += sectionNames["Example"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Example"])) + "\n\n"
		content += story.Example + "\n\n"
	}

	if story.Talk != "" {
		content += sectionNames["Interaction"] + "\n"
		content += strings.Repeat("-", len(sectionNames["Interaction"])) + "\n\n"
		content += story.Talk + "\n"
	}

	return strings.TrimSpace(content)
}

func (w TaskWriter) Archive() error {
	err := w.CreateDir("archive")
	if err != nil {
		msg := "create archive directory"
		return etrace.Wrap(msg, err)
	}

	for _, file := range w.task.Archive.Files {
		// Create necessary subdirectories
		dir := filepath.Dir(file.RelPath)
		if dir != "." {
			archiveSubDir := filepath.Join("archive", dir)
			err := w.CreateDirAll(archiveSubDir)
			if err != nil {
				msg := fmt.Sprintf("create archive subdirectory %s", dir)
				return etrace.Wrap(msg, err)
			}
		}

		filePath := filepath.Join("archive", file.RelPath)
		err := w.WriteFile(filePath, file.Content)
		if err != nil {
			msg := fmt.Sprintf("write archive file %s", file.RelPath)
			return etrace.Wrap(msg, err)
		}
	}

	return nil
}

func (w TaskWriter) CreateDirAll(path string) error {
	absPath := filepath.Join(w.path, path)
	err := os.MkdirAll(absPath, 0755)
	if err != nil {
		msg := fmt.Sprintf("create directory %s", path)
		return etrace.Wrap(msg, err)
	}
	return nil
}
