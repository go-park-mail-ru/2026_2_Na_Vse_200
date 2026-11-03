package migration

import (
	"errors"
	"fmt"
	"strconv"
)

const usage = "usage: migrate up | version | down <positive steps>"

type command struct {
	action string
	steps  int
}

func parseCommand(args []string) (command, error) {
	if len(args) == 0 {
		return command{}, errors.New(usage)
	}
	cmd := command{action: args[0]}
	switch cmd.action {
	case "-h", "--help", "up", "version":
		if len(args) != 1 {
			return command{}, fmt.Errorf("%s does not accept arguments", cmd.action)
		}
	case "down":
		if len(args) != 2 {
			return command{}, errors.New("usage: migrate down <positive steps>")
		}
		steps, err := strconv.Atoi(args[1])
		if err != nil || steps <= 0 {
			return command{}, errors.New("down requires a positive number of steps")
		}
		cmd.steps = steps
	default:
		return command{}, fmt.Errorf("unknown command %q", cmd.action)
	}
	return cmd, nil
}
