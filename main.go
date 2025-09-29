package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/common/zips"
	"github.com/spf13/cobra"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		wd, _ := os.Getwd()
		fmt.Printf("Warning: .env file not found in %s\n", wd)
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
				errorr("%s\n", etrace.GetDebugStr(err))
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
				if etrace.IsCritical(err) {
					errorr("%s\n", etrace.GetDebugStr(err))
				} else {
					warn("%s\n", etrace.GetDebugStr(err))
				}
			}
		},
	}

	var assistCmd = &cobra.Command{
		Use:   "assist [task-path]",
		Short: "Assist filling out missing info using AI",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := assistFunc(args[0]); err != nil {
				errorr("%s\n", etrace.GetDebugStr(err))
			}
		},
	}

	var rewriteZip bool
	var rewriteCmd = &cobra.Command{
		Use:   "rewrite [task-path]",
		Short: "Rewrite a task in-place (dir or .zip)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := rewrite(args[0], rewriteZip); err != nil {
				if etrace.IsCritical(err) {
					errorr("%s\n", etrace.GetDebugStr(err))
				} else {
					warn("%s\n", etrace.GetDebugStr(err))
				}
			}
		},
	}

	rewriteCmd.Flags().BoolVar(&rewriteZip, "zip", false, "Rewrite a .zip task (path must be a .zip)")

	rootCmd.AddCommand(transformCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(assistCmd)
	rootCmd.AddCommand(rewriteCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// extractToTmpIfZip accepts either a directory path or a .zip path. In case of a zip,
// it unzips it into a temporary directory and returns the directory containing
// the task root. It supports zips that either contain the task at root or inside
// a single top-level directory. The caller must call the returned cleanup func.
func extractToTmpIfZip(src string) (string, func(), error) {
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
