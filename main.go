package main

import (
	"log"
	"os"

	"eve.evalgo.org/cli"
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
