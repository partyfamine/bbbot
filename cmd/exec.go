package cmd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"

	"github.com/partyfamine/bbbot/bot"
	"github.com/spf13/cobra"
)

var (
	Exec = &cobra.Command{
		Use:   "exec",
		Short: "executes bot",
		Long:  "executes bot",
		Run:   exec,
	}

	brands = struct {
		nvidia   string
		evga     string
		pny      string
		msi      string
		gigabyte string
		asus     string
	}{
		nvidia:   "nvidia",
		evga:     "evga",
		pny:      "pny",
		msi:      "msi",
		gigabyte: "gigabyte",
		asus:     "asus",
	}
	gpuModel        string
	brand           string
	version         string
	sku             string
	jsonFileName    string
	paylpalEmail    string
	paylpalPassword string
	bestbuyEmail    string
	bestbuyPassword string
	isTest          bool
	headless        bool
	limit           float32
	remainingFunds  float32
)

func init() {
	Exec.Flags().StringVarP(&gpuModel, "gpu", "g", "", "gpu model you wish to purchase")
	Exec.Flags().StringVarP(&brand, "brand", "b", "", "brand of the gpu you wish to purchase")
	Exec.Flags().StringVarP(&version, "version", "v", "", "version of the gpu you wish to purchase (used to differentiate multiples of the same brand/model)")
	Exec.Flags().StringVarP(&sku, "sku", "s", "", "sku of the gpu you wish to purchase")
	Exec.Flags().StringVarP(&jsonFileName, "json", "j", "", "json file containing gpus to purchase")
	Exec.Flags().Float32VarP(&limit, "limit", "l", 0, "limit in usd of gpus to purchase")
	Exec.Flags().StringVar(&paylpalEmail, "paypal-email", "", "paypal email")
	Exec.Flags().StringVar(&paylpalPassword, "paypal-password", "", "paypal password")
	Exec.Flags().StringVar(&bestbuyEmail, "bestbuy-email", "", "bestbuy email")
	Exec.Flags().StringVar(&bestbuyPassword, "bestbuy-password", "", "bestbuy password")
	Exec.Flags().BoolVar(&isTest, "test", false, "will not confirm any orders if true")
	Exec.Flags().BoolVar(&isTest, "headless", false, "will not confirm any orders if true")

	Exec.MarkFlagRequired("paypal-email")
	Exec.MarkFlagRequired("paypal-password")
	Exec.MarkFlagRequired("bestbuy-email")
	Exec.MarkFlagRequired("bestbuy-password")
}

func exec(cmd *cobra.Command, args []string) {
	if limit != 0 {
		remainingFunds = limit
	}

	if jsonFileName != "" {
		jsonBytes, err := ioutil.ReadFile(jsonFileName)
		if err != nil {
			log.Fatal(err)
		}
		gpus := make([]GPU, 0)
		if err := json.Unmarshal(jsonBytes, &gpus); err != nil {
			log.Fatal(err)
		}

		skus := make([]string, len(gpus), len(gpus))
		for i, gpu := range gpus {
			if gpu.Sku != "" {
				skus[i] = gpu.Sku
				continue
			}
			foundSku, ok := skuMap[gpu]
			if !ok {
				log.Fatalf("invalid gpu format: %#v\n", gpu)
			}
			skus[i] = foundSku
		}

		log.Printf("launching bots for skus: %v\n", skus)

		ctx, cancel := context.WithCancel(context.Background())
		handleInterrupt(cancel)
		var wg sync.WaitGroup
		wg.Add(len(skus))
		for _, skuID := range skus {
			go func(skuID string) {
				b := bot.Bot{
					Sku:             skuID,
					Limit:           limit,
					RemainingFunds:  &remainingFunds,
					BestbuyEmail:    bestbuyEmail,
					BestbuyPassword: bestbuyPassword,
					PaylpalEmail:    paylpalEmail,
					PaylpalPassword: paylpalPassword,
					IsTest:          isTest,
					Headless:        headless,
				}
				b.Exec(ctx)
				wg.Done()
			}(skuID)
		}
		wg.Wait()
	} else {
		skuID := sku
		if skuID == "" {
			gpu := GPU{
				Brand:   brand,
				Model:   gpuModel,
				Version: version,
			}
			foundSku, ok := skuMap[gpu]
			if !ok {
				log.Fatal("gpu not supported")
			}
			skuID = foundSku
		}

		ctx, cancel := context.WithCancel(context.Background())
		handleInterrupt(cancel)
		b := bot.Bot{
			Sku:             skuID,
			Limit:           limit,
			RemainingFunds:  &remainingFunds,
			BestbuyEmail:    bestbuyEmail,
			BestbuyPassword: bestbuyPassword,
			PaylpalEmail:    paylpalEmail,
			PaylpalPassword: paylpalPassword,
			IsTest:          isTest,
			Headless:        headless,
		}
		b.Exec(ctx)
	}
}
