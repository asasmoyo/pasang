package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/otiai10/copy"
)

func executeLocal(conf config) error {
	info, err := os.Stat(conf.Source)
	if err != nil {
		return errors.New("can't find source file / folder")
	}

	if err := executeLocalCmd(conf.Source, conf.BeforeRunCmd); err != nil {
		return errors.New("before run commands return non 0 exit code")
	}

	releaseDir := filepath.Join(conf.Dest, "releases")
	if err := os.MkdirAll(releaseDir, info.Mode()); err != nil {
		return errors.New("failed to create release folder")
	}

	log.Println("copying source to release folder")
	targetDir := filepath.Join(conf.Dest, "releases", fmt.Sprintf("%d", time.Now().Unix()))
	if err := copy.Copy(conf.Source, targetDir); err != nil {
		return errors.New("failed copying source to release folder")
	}

	if err := executeLocalCmd(conf.Source, conf.BeforeRunCmd); err != nil {
		return errors.New("after run commands return non 0 exit code")
	}

	err = deleteOldReleases(releaseDir, conf.Keep)
	if err != nil {
		return errors.New("failed to deleted old release folders")
	}

	err = symlinkLocal(filepath.Join(conf.Dest, "current"), targetDir)
	if err != nil {
		return errors.New("failed to link release folder")
	}
	return nil
}

func executeLocalCmd(dir string, cmds []string) error {
	for _, cmd := range cmds {
		log.Printf("executing \"%s\" at %s\n", cmd, dir)

		parts := strings.Split(cmd, " ")
		name := parts[0]
		args := []string{}
		if len(parts) > 1 {
			args = parts[1:]
		}

		c := exec.Command(name, args...)
		c.Dir = dir
		out, err := c.CombinedOutput()
		log.Println(string(out))
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteOldReleases(dir string, keep int) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	var dirs []string
	for _, info := range infos {
		dirs = append(dirs, filepath.Join(dir, info.Name()))
	}
	sort.Strings(dirs)
	for i := 0; len(dirs) > keep+i; i++ {
		log.Printf("deleting old release: %s\n", dirs[i])
		if err := os.RemoveAll(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}

func symlinkLocal(target, source string) error {
	os.Remove(target)
	return os.Symlink(source, target)
}
