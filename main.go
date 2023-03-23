package main

import (
	"e2e-k8s/cmd"
	"os"
)

func main() {
	v := cmd.NewRootCmd(os.Stdout)
	if err := v.Execute(); err != nil {
		os.Exit(1)
	}

}
