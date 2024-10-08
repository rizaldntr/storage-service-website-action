package types

import "strings"

type ObjectACL string

func (o *ObjectACL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if strings.ToLower(s) == "private" {
		*o = PrivateACL
	} else {
		*o = PublicACL
	}
	return nil
}

const (
	PrivateACL ObjectACL = "private"
	PublicACL  ObjectACL = "public"
)
