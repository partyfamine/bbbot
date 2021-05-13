package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
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
	limit           float32
	remainingFunds  float32
	mu              sync.Mutex
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
	Exec.Flags().StringVar(&bestbuyEmail, "bestbuy-email", "e", "bestbuy email")
	Exec.Flags().StringVar(&bestbuyPassword, "bestbuy-password", "", "bestbuy password")
	Exec.Flags().BoolVar(&isTest, "test", false, "will not confirm any orders if true")
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
				execBot(ctx, skuID)
				wg.Done()
			}(skuID)
		}
		wg.Wait()
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		handleInterrupt(cancel)
		execBot(ctx, sku)
	}
}

func execBot(parentCtx context.Context, skuID string) {
	url := fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", skuID)
	allocateCtx, cancelAllocate := chromedp.NewExecAllocator(parentCtx, opts...)
	defer cancelAllocate()
	ctx, cancel := chromedp.NewContext(allocateCtx)
	defer cancel()

	navigateToPage(ctx, url)

	for {
		inStock := isInStock(ctx, skuID)
		if inStock {
			added := addToCart(ctx, skuID)
			if !added {
				continue
			}
			if inStock {
				break
			}
		}
	}

	if !withinPriceRange(ctx, ".priceView-customer-price span") {
		log.Printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", remainingFunds)
		return
	}
	login(ctx)
	if !withinPriceRange(ctx, ".price-summary__total-value") {
		log.Printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", remainingFunds)
		return
	}
	payWithPaypal(ctx, skuID)
	time.Sleep(30 * time.Second) //extra time to finish processing just in case
}
