package foundation

import (
	"errors"
	"testing"
)

type greeter interface{ Greet() string }

type english struct{}

func (english) Greet() string { return "hello" }

func TestContainerBindMake(t *testing.T) {
	c := NewContainer()
	if Bound[greeter](c) {
		t.Fatal("nothing should be bound yet")
	}
	if _, err := Make[greeter](c); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("want ErrServiceNotFound, got %v", err)
	}

	Bind[greeter](c, english{})
	g, err := Make[greeter](c)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if g.Greet() != "hello" {
		t.Fatalf("Greet = %q", g.Greet())
	}

	// Interface and concrete types are distinct keys.
	if Bound[english](c) {
		t.Fatal("concrete type should not be bound by interface binding")
	}
}

func TestContainerRebindReplaces(t *testing.T) {
	c := NewContainer()
	Bind(c, 1)
	Bind(c, 2)
	if v, _ := Make[int](c); v != 2 {
		t.Fatalf("rebind should replace, got %d", v)
	}
}

func TestContainerNilInterfaceBinding(t *testing.T) {
	c := NewContainer()
	Bind[greeter](c, nil)
	g, err := Make[greeter](c)
	if err != nil || g != nil {
		t.Fatalf("Make = %v, %v", g, err)
	}
}
