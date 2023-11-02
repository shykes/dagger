package main

import (
	"context"
	"fmt"
	"strings"

	"dagger.io/dagger"
	"github.com/spf13/cobra"
)

var printCmd = &FuncCommand{
	Name:  "print",
	Short: "print the output of a pipeline",
	Long:  "run a pipeline and print the result of its last function",
	OnSelectObjectLeaf: func(c *FuncCommand, name string) error {
		switch name {
		case Service:
			c.Select("id")
		case Container:
			c.Select("id")
		case Directory:
			c.Select("id")
		case File:
			c.Select("id")
		case Secret:
			c.Select("id")
		}
		return nil
	},
	AfterResponse: func(c *FuncCommand, cmd *cobra.Command, _ *modTypeDef, response any) error {
		return prettyPrint(c, cmd, response)
	},
}

func printDirectory(ctx context.Context, dag *dagger.Client, ID string) error {
	dir := dag.LoadDirectoryFromID(dagger.DirectoryID(ID))
	entries, err := dir.Entries(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", entries)
	return nil
}

func printContainer(ctx context.Context, dag *dagger.Client, ID string) error {
	ctr := dag.LoadContainerFromID(dagger.ContainerID(ID))
	// Entrypoint
	entrypoint, err := ctr.Entrypoint(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("- Entrypoint: %v\n", entrypoint)
	// Default args
	defaultArgs, err := ctr.DefaultArgs(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("- Default arguments: %v\n", defaultArgs)
	// Platform
	platform, err := ctr.Platform(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("- Platform: %s\n", platform)
	// Environment variables
	envVariables, err := ctr.EnvVariables(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("- Environment:\n")
	for _, kv := range envVariables {
		k, err := kv.Name(ctx)
		if err != nil {
			return err
		}
		v, err := kv.Value(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("\t%s\t%s\n", k, v)
	}
	return nil
}

func prettyPrint(c *FuncCommand, cmd *cobra.Command, response any) error {
	ctx := cmd.Context()
	if list, ok := (response).([]any); ok {
		cmd.Printf("[%d objects]:\n", len(list))
		for i, v := range list {
			cmd.Printf("%d/%d\n", i+1, len(list))
			prettyPrint(c, cmd, v)
		}
		return nil
	}
	dag := c.c.Dagger()
	// Look for IDs
	if ID, ok := (response).(string); ok {
		if strings.HasPrefix(ID, "core.") {
			parts := strings.SplitN(ID, ":", 2)
			if len(parts) >= 1 {
				switch parts[0] {
				case "core.Directory":
					return printDirectory(ctx, dag, ID)
				case "core.Service":
					return nil
				case "core.Secret":
					return nil
				case "core.Container":
					return printContainer(ctx, dag, ID)
				}
			}
		}
	}
	fmt.Printf("%+v\n", response)
	return nil
}
