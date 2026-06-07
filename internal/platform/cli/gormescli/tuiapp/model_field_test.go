package tuiapp

import (
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func capturedTUIModel(t *testing.T, model tea.Model) tui.Model {
	t.Helper()
	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}
	return m
}

func capturedRequiredTUIModelField[T any](t *testing.T, model tea.Model, name string) T {
	t.Helper()
	field := capturedRawTUIModelField(t, model, name)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(T)
}

func capturedOptionalTUIModelField[T any](t *testing.T, model tea.Model, name string) T {
	t.Helper()
	field := capturedRawTUIModelField(t, model, name)
	var zero T
	if field.IsNil() {
		return zero
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(T)
}

func capturedRawTUIModelField(t *testing.T, model tea.Model, name string) reflect.Value {
	t.Helper()
	m := capturedTUIModel(t, model)
	field := reflect.ValueOf(&m).Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("tui.Model missing %s field", name)
	}
	return field
}
