package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/external/lio/lio2023"
	"github.com/programme-lv/taskzip/external/lio/lio2024"
	"github.com/programme-lv/taskzip/taskfs"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "task-zip",
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

	rootCmd.AddCommand(transformCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func transform(src string, dst string, format string, zipOut bool) error {
	fmt.Printf("INFO:\trunning transform\n\t- src: %s\n\t- dst: %s\n\t- format: %s\n", src, dst, format)

	var task taskfs.Task
	var err error
	switch format {
	case "lio2023":
		task, err = lio2023.ParseLio2023TaskDir(src)
	case "lio2024":
		task, err = lio2024.ParseLio2024TaskDir(src)
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

	if err := taskfs.WriteZip(task, dst); err != nil {
		return etrace.Wrap("write zip", err)
	}
	fmt.Printf("INFO:\tsuccess; zip at %s\n", zipPath)
	fmt.Println("HINT:\tcheck out readme.md inside the .zip")
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
