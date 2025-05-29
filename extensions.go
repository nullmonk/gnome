package gnome

import (
	"syscall"
	"unsafe"

	"go.starlark.net/starlark"
	"golang.org/x/sys/unix"
)

// Custom functions only implemented by gnome (mostly as globals)

var exitCode int64

/* Exit the interpreter preventing execution of other scripts and exiting with the given status code */
func exit(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var code starlark.Int
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 0, &code); err != nil {
		return nil, err
	}
	exitCode, _ = code.Int64()
	thread.Cancel("user exit")
	return starlark.None, nil
}

/* Quit the starlark script. Does not prevent execution of other scripts */
func quit(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 0); err != nil {
		return nil, err
	}
	thread.Cancel("user quit")
	return starlark.None, nil
}

// Execveat executes at program at a path using a file descriptor.
// The go runtime process image is replaced by the executable described
// by the directory file descriptor and pathname.
func fexecveat(fd uintptr, pathname string, argv []string, envv []string, flags int) error {
	pathnamep, err := syscall.BytePtrFromString(pathname)
	if err != nil {
		return err
	}

	argvp, err := syscall.SlicePtrFromStrings(argv)
	if err != nil {
		return err
	}

	envvp, err := syscall.SlicePtrFromStrings(envv)
	if err != nil {
		return err
	}

	_, _, errno := syscall.Syscall6(
		unix.SYS_EXECVEAT,
		fd,
		uintptr(unsafe.Pointer(pathnamep)),
		uintptr(unsafe.Pointer(&argvp[0])),
		uintptr(unsafe.Pointer(&envvp[0])),
		uintptr(flags),
		0,
	)
	return errno
}

/*
// Stop execution of all the threads, then fexec a binary from either the system or the asset locker
func fallback(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var code starlark.String
	var cmdArgs *starlark.List
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 1, &code, &cmdArgs); err != nil {
		return nil, err
	}

	thread.Cancel("fallback")
	// Convert args to []string
	argsActual := make([]string, 0, cmdArgs.Len())
	iter := cmdArgs.Iterate()
	defer iter.Done()
	var x starlark.Value
	for iter.Next(&x) {
		i := x.(starlark.String).GoString()
		argsActual = append(argsActual, i)
	}

	assets := modules.GetAssetLocker()
	// First, if this matches an asset, call the asset
	if assets != nil {
		if f, err := assets.Open(code.GoString()); err == nil {
			defer f.Close()
			flag := unix.MFD_CLOEXEC
			// if close-on-exec flag has been set when fd points to a script,
			// then fexecve() fails with the error ENOENT. Peek this to undo if its a script
			// TODO undo this
			fd, err := unix.MemfdCreate("", flag)
			if err != nil {
				return nil, err
			}

			memfd := os.NewFile(uintptr(fd), code.GoString())
			if _, err = io.Copy(memfd, f); err != nil {
				return nil, err
			}
			return starlark.None, fexecveat(memfd.Fd(), "", argsActual, os.Environ(), unix.AT_EMPTY_PATH)
		}
	}

	// See if its a binary on disk
	f, err := os.Open(code.GoString())
	if err == nil {
		return starlark.None, fexecveat(f.Fd(), "", argsActual, os.Environ(), unix.AT_EMPTY_PATH)
	}
	return starlark.None, nil
}
*/
