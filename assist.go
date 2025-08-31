package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/programme-lv/taskzip/assist"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

func assistFunc(src string) error {
	info("running assist on %s", src)
	dir, cleanup, err := extractToTmpIfZip(src)
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
		"use .typ from archive to fill subtask descriptions",
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
	case "2":
		info("received answer: %s (filling subtask descriptions)", answer)
		task, err = fillSubtaskDescriptions(task)
		if err != nil {
			return etrace.Wrap("fill subtask descriptions", err)
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

func tempFile(pattern string) (*os.File, error) {
	os.MkdirAll(filepath.Join(os.TempDir(), "taskzip"), 0755)
	return os.CreateTemp(filepath.Join(os.TempDir(), "taskzip"), pattern)
}

func fillLvMdStatement(task taskfs.Task) (taskfs.Task, error) {
	// find .typ files in archive
	files := collectTypFiles(task)
	if len(files) != 1 {
		return task, etrace.NewError(fmt.Sprintf("expected 1 .typ file, got %d", len(files)))
	}
	imagesAvailable := []string{}
	for _, image := range task.Statement.Images {
		imagesAvailable = append(imagesAvailable, image.Fname)
	}
	prompt := fillLvMdStatementPrompt(string(files[0].Content), task.Testing.TestingT, imagesAvailable)

	promptFile, err := tempFile("prompt-*.txt")
	if err != nil {
		return task, etrace.Wrap("create prompt file", err)
	}
	defer promptFile.Close()
	_, err = promptFile.WriteString(prompt)
	if err != nil {
		return task, etrace.Wrap("write prompt", err)
	}
	info("prompt saved at: %s", promptFile.Name())

	response, err := assist.AskChatGptSimple(prompt)
	if err != nil {
		return task, etrace.Wrap("ask chat gpt", err)
	}

	responseFile, err := tempFile("response-*.md")
	if err != nil {
		return task, etrace.Wrap("create response file", err)
	}
	defer responseFile.Close()
	_, err = responseFile.WriteString(response)
	if err != nil {
		return task, etrace.Wrap("write response", err)
	}
	info("response saved at: %s", responseFile.Name())

	story, err := taskfs.ParseMdStory(response, "lv")
	if err != nil {
		return task, etrace.Wrap("parse md story", err)
	}
	task.Statement.Stories["lv"] = story
	return task, nil
}

func fillLvMdStatementPrompt(typFile string, testingType string, imagesAvailable []string) string {
	prompt := `You are a precise technical writer. Return your final answer as RAW GitHub Flavored Markdown ONLY. Do NOT wrap in code fences. Do NOT include any prose before or after the markdown.

Your task is to transfer an competitive programming task statement from Typst (.typ) format to Markdown (.md) format. Language of the statement is Latvian.

The resulting markdown may contain mathematical expressions. Convert the math expressions to KaTeX-compatible format using dollar signs ($...$).
KaTeX use LaTeX math syntax and so "<=" is a an invalid expression. Use something like "leq" instead.

Don't use fancy UTF characters like emojis or symbols such as "«".

My markdown parser also does not support double (consecutive) dashes such as "--". If you need to use dashes, use single dash at a time.

`

	if testingType != "interactor" {
		prompt += `Result should contain only 3 sections: stāsts, ievaddati, izvaddati. Do not include any other information e.g. 'see constraints in contest system'. It should look like this with ... replaced with actual content:
'''
Stāsts
------

...

Ievaddati
---------

...

Izvaddati
---------

...
'''
`
	} else {
		prompt += `Result should contain only 3 sections: stāsts, komunikācija, peimērs. Do not include any other information e.g. 'see constraints in contest system' or 'piezīmes'. It should look like this with ... replaced with actual content:
'''
Stāsts
------

...

Komunikācija
---------

...

Piemērs
---------

...
'''

Additionally, sometimes there is also information on partial scoring. This should be added to the section "Vērtēšana".
Do not include subtask descriptions. Those will be extracted later. Just the partial scoring if there is any.

In my online judge, info on how to flush output in interactive tasks is automatically added to the statement. Do not include this information.
Auto-specified is also the behavior of the online judge if the query limit is exceeded if such a limit is present in the task.
Also don't include information on custom invocations as present in CMS (contest management system). I haven't implemented that yet.

Furthermore, i have developed a custom way to specify table-formed examples for interactive tasks. It looks like a codeblock with language specified as json.
Example (likely from a different task) (you can remove the comments column if there isn't one in the original) (code setting for column makes the column a font-mono code block):
'''json
{
"component": "table",
"cols": [
{"header": "Ievaddati", "code": true},
{"header": "Izvaddati (jūsu programmas vaicājumi)","code": true},
{"header": "Komentāri"}
],
"data": [
[
"8\n2 5\n8 7\n4 2\n1 8\n3 6\n3 7\n8 2\n",
"",
"Atbilst tekstā dotajam attēlam."
],
[
"",
"0 1 6",
""
],
[
"1",
"",
""
],
[
"",
"0 7 5",
""
],
[
"1",
"",
""
],
[
"",
"1 4 2",
"Bojātais kabelis var būt vienīgi starp $2$ un $4$."
]
]
}
'''
Note that JSON does not support multiline strings. If it's needed, use escaped newline.

`
	}

	prompt += `
Here is the Typst file:

'''typst
%s
'''
`
	if len(imagesAvailable) > 0 {
		prompt += `
It is possible to include images in the statement. Like this: ![2. attēls: Adapteru izmēri](1.png).
Available images: %s. It's fine to number the images yourself. They don't automatically get numbered.
If you won't add a number to an image, i'll have to do it myself.
`
	}

	prompt = strings.ReplaceAll(prompt, "'''", "```")

	return fmt.Sprintf(prompt, typFile, imagesAvailable)
}

func fillSubtaskDescriptions(task taskfs.Task) (taskfs.Task, error) {
	n := len(task.Statement.Subtasks)
	if n == 0 {
		return task, etrace.NewError("no subtasks to fill")
	}
	files := collectTypFiles(task)
	prompt := fillSubtaskDescriptionsPrompt(n, string(files[0].Content))
	resp, err := assist.AskChatGptSimple(prompt)
	if err != nil {
		return task, etrace.Wrap("ask chat gpt", err)
	}
	arr, err := parseJsonArr(resp)
	if err != nil {
		return task, etrace.Wrap("parse json", err)
	}
	if len(arr) != n {
		msg := fmt.Sprintf("expected %d descriptions, got %d", n, len(arr))
		return task, etrace.NewError(msg)
	}
	for i := 0; i < n; i++ {
		if task.Statement.Subtasks[i].Desc == nil {
			task.Statement.Subtasks[i].Desc = make(taskfs.I18N[string])
		}
		task.Statement.Subtasks[i].Desc["lv"] = strings.TrimSpace(arr[i])
	}
	return task, nil
}

func collectTypFiles(task taskfs.Task) []assist.File {
	files := []assist.File{}
	for _, file := range task.Archive.Files {
		if strings.HasSuffix(file.RelPath, ".typ") || strings.HasSuffix(file.RelPath, ".txt") {
			files = append(files, assist.File{
				Content: file.Content,
				Fname:   file.RelPath,
			})
		}
	}
	return files
}

func fillSubtaskDescriptionsPrompt(noOfSubtasks int, typstFile string) string {
	p := `
You are a precise technical writer.
Return RAW JSON ONLY: an array of strings of length %d. No prose, no code fences.

Task: extract concise Latvian descriptions for each subtask (1..%d). 
No numbering, just the text. Keep very close to original text.

Subtask descriptions should be markdown one-liners.
KaTeX-compatible math expressions are allowed in dollar signs ($...$).

The following is a Typst (.typ) file to import descriptions from.

%s
`
	typstFile = fmt.Sprintf("```typst\n%s\n```", typstFile)
	return fmt.Sprintf(p, noOfSubtasks, noOfSubtasks, typstFile)
}

func parseJsonArr(s string) ([]string, error) {
	var arr []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &arr); err != nil {
		return nil, etrace.Trace(err)
	}
	return arr, nil
}
