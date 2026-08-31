package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestCLIHelp(t *testing.T) {
	var buf bytes.Buffer
	c := New(&buf)
	c.Register(&Command{
		Name:  "sessions",
		Short: "list active sessions",
		Run:   func(args []string) error { return nil },
	})
	if err := c.Run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected help output")
	}
	if !bytes.Contains([]byte(out), []byte("sessions")) {
		t.Fatalf("expected sessions in help output, got: %s", out)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	c := New(nil)
	err := c.Run([]string{"doesnotexist"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLIRunCommand(t *testing.T) {
	var buf bytes.Buffer
	c := New(&buf)
	called := false
	c.Register(&Command{
		Name:  "ping",
		Short: "ping the team server",
		Run: func(args []string) error {
			called = true
			return nil
		},
	})
	if err := c.Run([]string{"ping"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected ping command to be called")
	}
}

func TestCLICommandError(t *testing.T) {
	c := New(nil)
	c.Register(&Command{
		Name:  "fail",
		Short: "always fails",
		Run:   func(args []string) error { return errors.New("intentional failure") },
	})
	if err := c.Run([]string{"fail"}); err == nil {
		t.Fatal("expected error from fail command")
	}
}
