package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type Command struct {
	Name    string
	Short   string
	Flags   *flag.FlagSet
	Run     func(args []string) error
}

type CLI struct {
	commands map[string]*Command
	out      io.Writer
}

func New(out io.Writer) *CLI {
	if out == nil {
		out = os.Stdout
	}
	return &CLI{
		commands: map[string]*Command{},
		out:      out,
	}
}

func (c *CLI) Register(cmd *Command) {
	if cmd.Flags == nil {
		cmd.Flags = flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	}
	cmd.Flags.SetOutput(c.out)
	c.commands[cmd.Name] = cmd
}

func (c *CLI) Run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		c.printUsage()
		return nil
	}
	name := args[0]
	cmd, ok := c.commands[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	if err := cmd.Flags.Parse(args[1:]); err != nil {
		return err
	}
	return cmd.Run(cmd.Flags.Args())
}

func (c *CLI) printUsage() {
	fmt.Fprintln(c.out, "hsc2 operator CLI")
	fmt.Fprintln(c.out, "")
	fmt.Fprintln(c.out, "Usage:")
	fmt.Fprintln(c.out, "  hsc2 <command> [flags] [args]")
	fmt.Fprintln(c.out, "")
	fmt.Fprintln(c.out, "Commands:")
	for _, cmd := range c.commands {
		fmt.Fprintf(c.out, "  %-20s %s\n", cmd.Name, cmd.Short)
	}
	fmt.Fprintln(c.out, "")
	fmt.Fprintln(c.out, "Run 'hsc2 <command> --help' for command-specific flags.")
}
