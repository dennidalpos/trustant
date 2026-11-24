// Copyright 2025-2026 Nuvolaris Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package main

import (
	"errors"
	"os"
	"path/filepath"
)

var errUnsafeWrite = errors.New("refusing to write outside the workbench or through a symlink")

// openWorkbenchFile opens rel (a slash-separated path below root) for writing
// without following a symlink that leaves the workbench. The workbench is
// writable by the coding assistant, so a generated symlink named AGENTS.md or
// upload/<file> must not turn a UI save into a write to ~/.ssh or a host .env.
//
// The parent directory is resolved and must stay inside the resolved root; the
// final component must not be a symlink, and the open itself refuses one
// (O_NOFOLLOW) so a link swapped in after the check is also rejected.
func openWorkbenchFile(root, rel string, flag int, perm os.FileMode) (*os.File, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return nil, errUnsafeWrite
	}
	resolvedDir, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, filepath.Dir(clean)))
	if err != nil {
		return nil, err
	}
	if !withinRoot(resolvedRoot, resolvedDir) {
		return nil, errUnsafeWrite
	}
	target := filepath.Join(resolvedDir, filepath.Base(clean))
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
			return nil, errUnsafeWrite
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return os.OpenFile(target, flag|noFollowFlag, perm)
}

// writeWorkbenchFile replaces the content of rel below root, creating it when
// missing, under the same containment rules as openWorkbenchFile.
func writeWorkbenchFile(root, rel string, data []byte, perm os.FileMode) error {
	file, err := openWorkbenchFile(root, rel, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// writeFileNoFollow is writeWorkbenchFile for a file that sits directly in its
// workbench directory (.env, AGENTS.md, .gitignore, ...): the containing
// directory is the root, so only the final component can be a planted link.
func writeFileNoFollow(path string, data []byte, perm os.FileMode) error {
	return writeWorkbenchFile(filepath.Dir(path), filepath.Base(path), data, perm)
}
