package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var List = &cobra.Command{
	Use:   "list",
	Short: "lists available gpus",
	Long:  "lists available gpus",
	Run:   list,
}

func list(cmd *cobra.Command, args []string) {
	gpus := make([]GPU, len(skuMap), len(skuMap))
	i := 0
	for gpuKey := range skuMap {
		gpus[i] = gpuKey
		i++
	}

	sort.Slice(gpus, func(i, j int) bool {
		if gpus[i].Model == gpus[j].Model {
			return gpus[i].Brand > gpus[j].Brand
		}
		return gpus[i].Model > gpus[j].Model
	})

	for _, gpu := range gpus {
		versionStr := ""
		if gpu.Version != "" {
			versionStr = fmt.Sprintf(", version: %s", gpu.Version)
		}
		fmt.Printf("model: %s, brand: %s%s\n", gpu.Model, gpu.Brand, versionStr)
	}
}
