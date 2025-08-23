package main

import (
	"os"
	"path/filepath"

	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

func rewrite(src string, zipOut bool) error {
	info("running rewrite\n\t- src: %s\n\t- out: %s", src, map[bool]string{true: "zip", false: "dir"}[zipOut])

	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return etrace.Wrap("abs src", err)
	}

	srcDir, cleanup, err := extractToTmpIfZip(srcAbs)
	if err != nil {
		return etrace.Wrap("prepare src", err)
	}
	defer cleanup()

	t, err := taskfs.Read(srcDir)
	if err != nil {
		return etrace.Wrap("read task", err)
	}
	if err := t.Validate(); err != nil {
		if etrace.IsCritical(err) {
			return etrace.Wrap("validate task", err)
		}
	}
	info("task read successfully")
	printTaskOverview(t)

	parent := filepath.Dir(srcAbs)
	tmpParent, err := os.MkdirTemp(parent, ".taskzip-rewrite-")
	if err != nil {
		return etrace.Wrap("mktemp in parent", err)
	}
	defer os.RemoveAll(tmpParent)

	var tmpOut string
	if zipOut {
		tmpOut = filepath.Join(tmpParent, t.ShortID+".zip")
		if err := taskfs.WriteZip(t, tmpOut); err != nil {
			return etrace.Wrap("write tmp zip", err)
		}
	} else {
		tmpOut = filepath.Join(tmpParent, t.ShortID)
		if err := taskfs.Write(t, tmpOut); err != nil {
			return etrace.Wrap("write tmp dir", err)
		}
	}

	finalPath := filepath.Join(parent, t.ShortID)
	if zipOut {
		finalPath = finalPath + ".zip"
	}

	// remove original path
	if err := os.RemoveAll(srcAbs); err != nil {
		return etrace.Wrap("remove original", err)
	}

	// if a different target already exists, remove it too
	if finalPath != srcAbs {
		if _, err := os.Stat(finalPath); err == nil {
			warn("removing existing target %s", finalPath)
			if err := os.RemoveAll(finalPath); err != nil {
				return etrace.Wrap("remove existing target", err)
			}
		}
	}

	if err := os.Rename(tmpOut, finalPath); err != nil {
		return etrace.Wrap("rename into place", err)
	}

	if zipOut {
		info("rewritten; zip at %s", finalPath)
		return nil
	}

	info("rewritten; dir at %s", finalPath)
	return nil
}
