package main

import (
	"errors"
	"log"
	"os"

	"github.com/PwzXxm/raft-lite/functests"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/urfave/cli"
)

func main() {
	cmd_simulation := cli.Command{
		Name:  "simulation",
		Usage: "commands for running simulation",
		Subcommands: []cli.Command{
			{
				Name:  "local",
				Usage: "start a local simulation",
				Flags: []cli.Flag{
					cli.Int64Flag{Name: "n", Usage: "number of peers", Required: true},
				},
				Action: func(c *cli.Context) error {
					// TODO: not sure why the [Required: true] above
					// isn't working.
					if c.Int("n") == 0 {
						return errors.New("please provide -n")
					}
					return local_simulation(c.Int("n"))
				},
			},
		},
	}
	cmd_functional := cli.Command{
		Name:  "functionaltest",
		Usage: "commands for running functional tests",
		Subcommands: []cli.Command{
			{
				Name:  "list",
				Usage: "list all avaliable tests",
				Action: func(c *cli.Context) error {
					functests.List()
					return nil
				},
			},
			{
				Name:  "count",
				Usage: "count all avaliable tests",
				Action: func(c *cli.Context) error {
					functests.Count()
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "run a specific tests",
				Flags: []cli.Flag{
					cli.Int64Flag{Name: "n", Usage: "test id", Required: true},
				},
				Action: func(c *cli.Context) error {
					return functests.Run(c.Int("n"))
				},
			},
		},
	}
	app := &cli.App{
		Commands: []cli.Command{
			cmd_simulation,
			cmd_functional,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func local_simulation(n int) error {
	rf := simulation.RunLocally(n)
	defer rf.StopAll()

	rf.StartReadingCMD()
	return nil
}
