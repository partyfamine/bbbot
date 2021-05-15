package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func parseURL() string {
	if sku == "" {
		gpuKey := GPU{Model: gpuModel, Brand: brand, Version: version}
		skuFromMap, ok := skuMap[gpuKey]
		if !ok {
			log.Fatal("gpu not supported")
		}
		sku = skuFromMap
	}

	return fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", sku)
}

func handleInterrupt(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
	}()
}

func withinPriceRange(ctx context.Context, priceSelector string) bool {
	if limit == 0 {
		return true
	}

	var priceStr string
	declineSurvey(ctx)
	err := chromedp.Run(ctx, chromedp.Text(priceSelector, &priceStr, chromedp.ByQuery))
	if err != nil {
		log.Println("price failure")
		log.Fatal(err)
	}

	mu.Lock()
	curFunds := remainingFunds
	mu.Unlock()

	gpuPrice := mustStrToPrice(priceStr)
	if err != nil {
		log.Fatal(err)
	}
	return float32(gpuPrice) < curFunds
}

func elementExists(ctx context.Context, path string) bool {
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx, chromedp.Nodes(path, &nodes, chromedp.AtLeast(0), chromedp.ByQuery)); err != nil {
		log.Fatal(err)
	}
	return len(nodes) > 0
}

func mustStrToPrice(str string) float32 {
	if str[0] == '$' {
		str = str[1:]
	}

	str = strings.ReplaceAll(str, ",", "")

	price, err := strconv.ParseFloat(str, 32)
	if err != nil {
		log.Fatal(err)
	}

	return float32(price)
}

func mustRun(ctx context.Context, actions ...chromedp.Action) {
	declineSurvey(ctx)
	err := chromedp.Run(ctx, actions...)
	if err != nil {
		log.Fatal(err)
	}
}

func mustRunWithSuccessfulResp(ctx context.Context, actions ...chromedp.Action) {

	for {
		declineSurvey(ctx)
		resp, err := chromedp.RunResponse(ctx, actions...)
		if err != nil {
			log.Fatal(err)
		}

		if resp != nil {
			break
		}
	}
}
