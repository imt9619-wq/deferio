package utils

import (
	"context"
	"reflect"
	"unsafe"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
)

func PrivateFieldByName[T any](s any, name string) (T, bool) {
	var t T
	if name == "" {
		return t, false
	}
	reflectField := reflect.ValueOf(t).Elem().FieldByName(name)
	if !reflectField.IsValid() {
		return t, false
	}
	return reflect.NewAt(reflectField.Type(), unsafe.Pointer(reflectField.UnsafeAddr())).Elem().Interface().(T), true
}

// noinspection ALL
//
//go:linkname SessionControlable github.com/df-mc/dragonfly/server/session.(*Session).withControllable
func SessionControlable(s *session.Session, ctx context.Context, f func(tx *world.Tx, c session.Controllable) error) error 