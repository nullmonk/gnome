package modules

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"go.starlark.net/starlark"
)

// Implement https://docs.realm.pub/user-guide/eldritch#assets

type AssetModule struct {
	locker fs.FS
	Module
}

func (a *AssetModule) assetsList(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 0); err != nil {
		return nil, err
	}
	if a.locker == nil {
		return starlark.None, fmt.Errorf("asset locker not initialized")
	}
	return ToStarlarkValue(a.GetAssets())
}

func NewAssetModule(locker fs.FS) *AssetModule {
	m := &AssetModule{
		locker: locker,
		Module: make(Module),
	}
	m.Module["copy"] = starlark.NewBuiltin("assets.copy", m.assetsCopy)
	m.Module["list"] = starlark.NewBuiltin("assets.list", m.assetsList)
	m.Module["read"] = starlark.NewBuiltin("assets.read", m.assetsRead)
	m.Module["read_binary"] = starlark.NewBuiltin("assets.read_binary", m.assetsReadBinary)
	return m
}

func (a *AssetModule) assetsCopy(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name starlark.String
	var dst starlark.String
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 2, &name, &dst); err != nil {
		return nil, err
	}
	if a.locker == nil {
		return starlark.None, fmt.Errorf("asset locker not initialized")
	}

	f, err := a.locker.Open(name.GoString())
	if err != nil {
		return starlark.None, err
	}

	d, err := os.Create(dst.GoString())
	if err != nil {
		return starlark.None, err
	}
	defer f.Close()
	_, err = io.Copy(d, f)
	return starlark.None, err
}

func (a *AssetModule) assetsRead(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name starlark.String
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 1, &name); err != nil {
		return nil, err
	}
	if a.locker == nil {
		return starlark.None, fmt.Errorf("asset locker not initialized")
	}

	f, err := a.locker.Open(name.GoString())
	if err != nil {
		return starlark.None, err
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return starlark.None, err
	}
	return starlark.String(string(buf)), nil
}

func (a *AssetModule) assetsReadBinary(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name starlark.String
	if err := starlark.UnpackPositionalArgs("", args, kwargs, 1, &name); err != nil {
		return nil, err
	}
	if a.locker == nil {
		return starlark.None, fmt.Errorf("asset locker not initialized")
	}

	f, err := a.locker.Open(name.GoString())
	if err != nil {
		return starlark.None, err
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return starlark.None, err
	}
	return starlark.Bytes(buf), nil
}

func (a *AssetModule) GetAssets() []string {
	assets := make([]string, 0, 64)
	fs.WalkDir(a.locker, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		assets = append(assets, path)
		return nil
	})
	return assets
}
