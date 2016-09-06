// Copyright 2016 The Linux Foundation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package image

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// ValidateLayout walks through the file tree given by src and
// validates the manifest pointed to by the given refs
// or returns an error if the validation failed.
func ValidateLayout(src string, refs []string, out *log.Logger) error {
	return validate(newPathWalker(src), refs, out)
}

// Validate walks through the given .tar file and
// validates the manifest pointed to by the given refs
// or returns an error if the validation failed.
func Validate(tarFile string, refs []string, out *log.Logger) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return validate(newTarWalker(f), refs, out)
}

func validate(w walker, refs []string, out *log.Logger) error {
	for _, r := range refs {
		ref, err := findDescriptor(w, r)
		if err != nil {
			return err
		}

		if err = ref.validate(w); err != nil {
			return err
		}

		m, err := findManifest(w, ref)
		if err != nil {
			return err
		}

		if err := m.validate(w); err != nil {
			return err
		}
		if out != nil {
			out.Printf("reference %q: OK", r)
		}
	}
	return nil
}

// UnpackLayout walks through the file tree given by src and
// using the layers specified in the manifest pointed to by the given ref
// and unpacks all layers in the given destination directory
// or returns an error if the unpacking failed.
func UnpackLayout(src, dest, ref string) error {
	return unpack(newPathWalker(src), dest, ref)
}

// Unpack walks through the given .tar file and
// using the layers specified in the manifest pointed to by the given ref
// and unpacks all layers in the given destination directory
// or returns an error if the unpacking failed.
func Unpack(tarFile, dest, ref string) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return unpack(newTarWalker(f), dest, ref)
}

func unpack(w walker, dest, refName string) error {
	ref, err := findDescriptor(w, refName)
	if err != nil {
		return err
	}

	if err = ref.validate(w); err != nil {
		return err
	}

	m, err := findManifest(w, ref)
	if err != nil {
		return err
	}

	if err = m.validate(w); err != nil {
		return err
	}

	return m.unpack(w, dest)
}

// CreateRuntimeBundleLayout walks through the file tree given by src and
// creates an OCI runtime bundle in the given destination dest
// or returns an error if the unpacking failed.
func CreateRuntimeBundleLayout(src, dest, ref, root string) error {
	return createRuntimeBundle(newPathWalker(src), dest, ref, root)
}

// CreateRuntimeBundle walks through the given .tar file and
// creates an OCI runtime bundle in the given destination dest
// or returns an error if the unpacking failed.
func CreateRuntimeBundle(tarFile, dest, ref, root string) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return createRuntimeBundle(newTarWalker(f), dest, ref, root)
}

func createRuntimeBundle(w walker, dest, refName, rootfs string) error {
	ref, err := findDescriptor(w, refName)
	if err != nil {
		return err
	}

	if err = ref.validate(w); err != nil {
		return err
	}

	m, err := findManifest(w, ref)
	if err != nil {
		return err
	}

	if err = m.validate(w); err != nil {
		return err
	}

	c, err := findConfig(w, &m.Config)
	if err != nil {
		return err
	}

	err = m.unpack(w, filepath.Join(dest, rootfs))
	if err != nil {
		return err
	}

	spec, err := c.runtimeSpec(rootfs)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dest, "config.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(spec)
}
