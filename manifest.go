// Copyright 2021 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"github.com/juju/systems/channel"
	"io"
	"io/ioutil"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/systems"
	"gopkg.in/yaml.v2"
)

// Manifest represents the recording of the building of the charm or bundle.
// The manifest file should represent the metadata.yaml, but a lot more
// information.
type Manifest struct {
	Bases []systems.Base `yaml:"bases"`
}

// Validate checks the manifest to ensure there are no empty names, nor channels,
// and that architectures are supported.
func (m *Manifest) Validate() error {
	for _, b := range m.Bases {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("invalid base: empty file")
		}
	}
	return nil
}

func (m *Manifest) UnmarshalYAML(f func(interface{}) error) error {
	raw := make(map[interface{}]interface{})
	err := f(&raw)
	if err != nil {
		return err
	}

	v, err := schema.List(baseSchema).Coerce(raw["bases"], nil)
	if err != nil {
		return errors.Annotatef(err, "coerce")
	}

	newV, ok := v.([]interface{})
	if !ok {
		return errors.Annotatef(err, "converting")
	}
	bases, err := parseBases(newV)
	if err != nil {
		return err
	}

	*m = Manifest{Bases: bases}
	return nil
}

func parseBases(input interface{}) ([]systems.Base, error) {
	var err error
	if input == nil {
		return nil, nil
	}
	res := []systems.Base(nil)
	for _, v := range input.([]interface{}) {
		base := systems.Base{}
		baseMap := v.(map[string]interface{})
		if value, ok := baseMap["name"]; ok {
			base.Name = value.(string)
		}
		if value, ok := baseMap["channel"]; ok {
			base.Channel, err = channel.Parse(value.(string))
			if err != nil {
				return nil, errors.Annotatef(err, "parsing channel %q", value.(string))
			}
		}
		err = base.Validate()
		if err != nil {
			return nil, errors.Trace(err)
		}
		res = append(res, base)
	}
	return res, nil
}

// ReadManifest reads in a Manifest from a charm's manifest.yaml.
// It is not validated at this point so that the caller can choose to override
// any validation.
func ReadManifest(r io.Reader) (*Manifest, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var manifest *Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, errors.Annotatef(err, "manifest")
	}
	if manifest == nil {
		return nil, errors.Annotatef(err, "invalid base in manifest")
	}
	return manifest, nil
}

var baseSchema = schema.FieldMap(
	schema.Fields{
		"name": schema.OneOf(
			schema.Const(systems.Ubuntu),
			schema.Const(systems.Windows),
			schema.Const(systems.CentOS),
			schema.Const(systems.OpenSUSE),
			schema.Const(systems.GenericLinux),
			schema.Const(systems.OSX),
		),
		"channel": schema.String(),
	}, schema.Defaults{
		"name":    schema.Omit,
		"channel": schema.Omit,
	})
