package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grovetools/tend/pkg/app"
	"github.com/grovetools/tend/pkg/harness"
)

func main() {
	scenarios := []*harness.Scenario{
		NotifySystemScenario(),
		NotifyNtfyScenario(),
	}

	if err := app.Execute(context.Background(), scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}