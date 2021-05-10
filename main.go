package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/chromedp/chromedp"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("must provide sku id as argument")
	}

	sku := os.Args[1] // 6429440 for RTX 3080
	url := fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", sku)
	fmt.Println(url)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	title := ""
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Title(&title))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(title)
}
