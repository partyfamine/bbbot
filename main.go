package main

import (
	"github.com/partyfamine/bbbot/cmd"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "bbbot",
	Short: "bot for purchasing gpus from best buy",
	Long:  "bot for purchasing gpus from best buy",
}

func init() {
	root.AddCommand(cmd.Exec)
	root.AddCommand(cmd.List)
}

func main() {
	root.Execute()
}
