package gnome

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/nullmonk/gnome/modules"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type Script struct {
	Name    string
	Src     any
	Globals starlark.StringDict
	Assets  fs.FS
}

type ErrorHandler func(script string, err error)
type PrintHandler func(script string, msg string)

// Parse the scripts to execute from the given asset FS. Useful for validating that the starlark is good before executing
func GetScripts(assets fs.FS) ([]*Script, error) {
	// Set the asset locker to whatever we have specified
	scripts_to_run := make([]*Script, 0, 1)
	// Loop through the assets dir an look for scripts to execute
	if assets == nil {
		return nil, fmt.Errorf("invalid asset locker")
	}
	err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".eldr") && !strings.HasSuffix(path, ".eldritch") {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		buf, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		_, err = syntax.LegacyFileOptions().Parse(path, buf, 0)
		if err != nil {
			return fmt.Errorf("invalid script: %s", err)
		}
		scripts_to_run = append(scripts_to_run, &Script{
			Name:   path,
			Src:    buf,
			Assets: assets})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scripts_to_run, nil
}

func Run(scripts []*Script, onprint PrintHandler, onerror ErrorHandler) {
	for _, s := range scripts {
		err := run(s, onprint)
		if err != nil {
			if onerror != nil {
				onerror(s.Name, err)
			} else {
				fmt.Fprintf(os.Stderr, "error executing %s: %s", s.Name, err)
			}
		}
	}
}

func run(script *Script, print PrintHandler) error {
	thread := &starlark.Thread{
		Name: script.Name,
	}
	if print != nil {
		thread.Print = func(thread *starlark.Thread, msg string) {
			print(thread.Name, msg)
		}
	}
	opts := &syntax.FileOptions{
		Set:             true,
		While:           true,
		TopLevelControl: true,
		GlobalReassign:  true,
		Recursion:       false,
	}

	libs := starlark.StringDict{
		"assets":  modules.NewAssetModule(script.Assets),
		"crypto":  &modules.Crypto,
		"file":    &modules.File,
		"http":    &modules.Http,
		"pivot":   &modules.Pivot,
		"process": &modules.Process,
		"regex":   &modules.Regex,
		"report":  &modules.Report,
		"sys":     &modules.Sys,
		"time":    &modules.Time,
		"exit":    starlark.NewBuiltin("exit", exit),
		"quit":    starlark.NewBuiltin("quit", quit),
		// TODO: Pprint
		//"fallback": starlark.NewBuiltin("fallback", fallback),
	}

	// Globals CAN overwrite the builtin libs, but we are going to assume the user intends that
	for k, v := range script.Globals {
		libs[k] = v
	}

	// Add the globals into the environment
	_, err := starlark.ExecFileOptions(opts, thread, script.Name, script.Src, libs)
	if err != nil {
		if e, ok := err.(*starlark.EvalError); ok {
			// Check what the error message is, that is how we determined if we quit or exited
			lines := strings.SplitN(e.Msg, ": ", 2)
			if len(lines) < 2 {
				return err
			}
			if lines[1] == "user exit" {
				// On exit calls, the interpreter also dies
				os.Exit(int(exitCode))
			} else if lines[1] == "user quit" {
				// on quit calls, only the script exits, not an error
				err = nil
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
