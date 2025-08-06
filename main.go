package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/programme-lv/task-zip/external/lio/lio2023"
	"github.com/programme-lv/task-zip/taskfs"
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

	var transformCmd = &cobra.Command{
		Use:   "transform",
		Short: "Transform task format to task-zip format",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("src: %s", src)
			log.Printf("dst: %s", dst)
			log.Printf("format: %s", format)
			err := transform(src, dst, format)
			if err != nil {
				log.Printf("Transform task failed: %v", err)
				os.Exit(1)
			}
			log.Print("Transform completed successfully")
		},
	}

	transformCmd.Flags().StringVarP(&src, "src", "s", "", "Source task directory path (*)")
	transformCmd.Flags().StringVarP(&dst, "dst", "d", "", "Destination parent dir where new task will be written (*)")
	transformCmd.Flags().StringVarP(&format, "format", "f", "", "Format of the import [lio2023] (*)")

	transformCmd.MarkFlagRequired("src")
	transformCmd.MarkFlagRequired("dst")
	transformCmd.MarkFlagRequired("format")

	rootCmd.AddCommand(transformCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func transform(src string, dst string, format string) error {
	if format != "lio2023" {
		return fmt.Errorf("unsupported format: %s", format)
	}

	fmt.Printf("Starting transformLio2023Task - src: %s, parent dst: %s\n", src, dst)

	task, err := lio2023.ParseLio2023TaskDir(src)
	if err != nil {
		return fmt.Errorf("error parsing task: %w", err)
	}
	fmt.Printf("Parsed LIO2023 task: %v\n", task.FullName)

	path := filepath.Join(dst, task.ShortID)
	fmt.Printf("Creating task directory: %s\n", path)

	err = taskfs.Write(task, path)
	if err != nil {
		return fmt.Errorf("error storing task: %w", err)
	}
	fmt.Println("Stored transformed task")

	return nil
}
