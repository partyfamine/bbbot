package cmd

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
)

func navigateToPage(ctx context.Context, url string) {
	title := ""
	log.Printf("navigating to %s\n", url)
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Title(&title))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(title)
}

func waitForStock(ctx context.Context) {
	var btnText string

	for {
		err := chromedp.Run(ctx,
			chromedp.Reload(),
			chromedp.Text(".add-to-cart-button", &btnText, chromedp.ByQuery))
		if err != nil {
			log.Fatal(err)
		}
		if btnText == "Add to Cart" {
			log.Println("in stock!!!")
			break
		} else {
			log.Printf("out of stock: %s\n", btnText)
		}

		if !withinPriceRange(ctx, ".priceView-customer-price span") {
			break
		}
	}
}

func addToCart(ctx context.Context) {
	log.Println("adding to cart")
	err := chromedp.Run(ctx, chromedp.Click(".add-to-cart-button", chromedp.ByQuery))
	if err != nil {
		log.Fatal(err)
	}

	if !elementExists(ctx, ".c-alert-content") {
		//TODO: screenshot
	}

	goToCart := ".go-to-cart-button"
	if !elementExists(ctx, goToCart) {
		goToCart = ".cart-link"
	}
	err = chromedp.Run(ctx, chromedp.Click(goToCart, chromedp.ByQuery))
	if err != nil {
		log.Fatal(err)
	}
}

func payWithPaypal(ctx context.Context) {
	log.Println("checking out via paypal")
	priceStr := ""
	err := chromedp.Run(ctx, chromedp.Text(".price-summary__total-value", &priceStr, chromedp.ByQuery))
	if err != nil {
		log.Fatal(err)
	}

	cartPrice := mustStrToPrice(priceStr)
	if err != nil {
		log.Fatal(err)
	}

	mu.Lock()
	remainingFunds -= float32(cartPrice)
	mu.Unlock()

	for {
		resp, err := chromedp.RunResponse(ctx, chromedp.Click(".checkout-buttons__paypal", chromedp.ByQuery))
		if err != nil {
			log.Fatal(err)
		}

		if resp != nil {
			break
		}
	}

	err = chromedp.Run(ctx,
		chromedp.SendKeys("email", email, chromedp.ByID),
		chromedp.Click("btnNext", chromedp.ByID))
	if err != nil {
		log.Fatal(err)
	}

	err = chromedp.Run(ctx,
		chromedp.SendKeys("password", password, chromedp.ByID),
		chromedp.Click("btnLogin", chromedp.ByID))
	if err != nil {
		log.Fatal(err)
	}

	for !elementExists(ctx, "[data-testid='pay-with']") {
	}
	_, err = chromedp.RunResponse(ctx, chromedp.Click("payment-submit-btn", chromedp.ByID))
	if err != nil {
		log.Fatal(err)
	}

	if !isTest {
		for !elementExists(ctx, ".button__fast-track") {
		}
		_, err = chromedp.RunResponse(ctx, chromedp.Click(".button__fast-track", chromedp.ByQuery))
		if err != nil {
			log.Fatal(err)
		}
	}
}
