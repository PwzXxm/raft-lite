/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 *   Minjian Chen 813534
 *   Shijie Liu   813277
 *   Weizhi Xu    752454
 *   Wenqing Xue  813044
 *   Zijun Chen   813190
 */

package main

import (
	"log"
	"os"

	"github.com/PwzXxm/raft-lite/client"
	"github.com/PwzXxm/raft-lite/functests"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func main() {
	// run simulation
	cmdSimulation := &cli.Command{
		Name:  "simulation",
		Usage: "commands for running simulation",
		Subcommands: []*cli.Command{
			{
				Name:  "local",
				Usage: "start a local simulation",
				Flags: []cli.Flag{
					&cli.Int64Flag{Name: "n", Usage: "number of peers", Required: true},
				},
				Action: func(c *cli.Context) error {
					if c.Int("n") == 0 {
						return errors.New("please provide -n")
					}
					return localSimulation(c.Int("n"))
				},
			},
		},
	}
	// run functional test
	cmdFunctional := &cli.Command{
		Name:  "functionaltest",
		Usage: "commands for running functional tests",
		Subcommands: []*cli.Command{
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
					&cli.Int64Flag{Name: "n", Usage: "test id", Required: true},
				},
				Action: func(c *cli.Context) error {
					return functests.Run(c.Int("n"))
				},
			},
		},
	}
	// run raft
	cmdStart := &cli.Command{
		Name:  "peer",
		Usage: "commands for running raft",
		Flags: []cli.Flag{
			&cli.PathFlag{Name: "c", Usage: "peer config file path", Required: true},
		},
		Action: func(c *cli.Context) error {
			return StartPeerFromFile(c.Path("c"))
		},
	}
	// run complex testcases where actions are generated randomly
	cmdIntegrationTest := &cli.Command{
		Name:  "integrationtest",
		Usage: "run complex testcases where actions are generated randomly",
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "t", Usage: "time in minutes", Required: true},
		},
		Action: func(c *cli.Context) error {
			return functests.RunComplex(c.Int64("t"))
		},
	}
	// run starting client
	cmdClient := &cli.Command{
		Name:  "client",
		Usage: "commands for starting client",
		Flags: []cli.Flag{
			&cli.PathFlag{Name: "c", Usage: "client config file path", Required: true},
		},
		Action: func(c *cli.Context) error {
			return startClient(c.Path("c"))
		},
	}
	app := &cli.App{
		Commands: []*cli.Command{
			cmdSimulation,
			cmdFunctional,
			cmdStart,
			cmdIntegrationTest,
			cmdClient,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func localSimulation(n int) error {
	rf := simulation.RunLocally(n)
	defer rf.StopAll()

	rf.StartReadingCMD()
	return nil
}

func startClient(filePath string) error {
	err := client.StartClientFromFile(filePath)
	return err
}
