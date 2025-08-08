package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/external/lio/lio2023"
	"github.com/programme-lv/task-zip/external/lio/lio2024"
	"github.com/programme-lv/task-zip/taskfs"
	"github.com/spf13/cobra"
)

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey && len(groups) == 0 {
					return slog.Attr{}
				}
				return a
			},
		}),
	))

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
			err := transform(src, dst, format)
			if err != nil {
				slog.Error("transform task failed", "error", err)
				// traced := errwrap.TracedError{}
				// if errors.As(err, &traced) {
				// 	fmt.Fprintln(os.Stderr, "trace:")
				// 	fmt.Fprintln(os.Stderr, traced.Trace())
				// }
				// os.Exit(1)
			}
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
	slog.Info("transform", "src", src, "dst", dst, "format", format)

	var task taskfs.Task
	var err error
	switch format {
	case "lio2023":
		task, err = lio2023.ParseLio2023TaskDir(src)
	case "lio2024":
		task, err = lio2024.ParseLio2024TaskDir(src)
	default:
		msg := fmt.Sprintf("unsupported task format: %s", format)
		return errwrap.Error(msg)
	}
	if err != nil {
		return errwrap.Wrap("parsing task in transform cmd", err)
	}

	if err := task.Validate(); err != nil {
		if errwrap.IsCritical(err) {
			msg := "validate task parsed"
			return errwrap.Wrap(msg, err)
		} else {
			slog.Warn("validation warnings", "error", err)
		}
	}

	path := filepath.Join(dst, task.ShortID)
	slog.Info("task parsed; write task dir", "path", path)

	err = taskfs.Write(task, path)
	if err != nil {
		msg := fmt.Sprintf("write task to %s", path)
		return errwrap.Wrap(msg, err)
	}

	return nil
}
