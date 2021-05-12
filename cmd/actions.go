package cmd

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
)

func navigateToPage(ctx context.Context, url string) {
	title := ""
	log.Printf("navigating to %s\n", url)
	mustRun(ctx,
		chromedp.Navigate(url),
		chromedp.Title(&title))
	log.Println(title)
}

func waitForStock(ctx context.Context) {
	var btnText string

	for {
		mustRun(ctx,
			chromedp.Reload(),
			chromedp.Text(".add-to-cart-button", &btnText, chromedp.ByQuery))

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
	mustRun(ctx, chromedp.Click(".add-to-cart-button", chromedp.ByQuery))

	if !elementExists(ctx, ".c-alert-content") {
		//TODO: screenshot
	}

	goToCart := ".go-to-cart-button"
	if !elementExists(ctx, goToCart) {
		goToCart = ".cart-link"
	}
	mustRun(ctx, chromedp.Click(goToCart, chromedp.ByQuery))
}

func payWithPaypal(ctx context.Context) {
	log.Println("checking out via paypal")
	priceStr := ""
	mustRun(ctx, chromedp.Text(".price-summary__total-value", &priceStr, chromedp.ByQuery))

	cartPrice := mustStrToPrice(priceStr)
	mu.Lock()
	remainingFunds -= float32(cartPrice)
	mu.Unlock()

	mustRunWithSuccessfulResp(ctx, chromedp.Click(".checkout-buttons__paypal", chromedp.ByQuery))

	mustRun(ctx,
		chromedp.SendKeys("email", email, chromedp.ByID),
		chromedp.Click("btnNext", chromedp.ByID))

	mustRun(ctx,
		chromedp.SendKeys("password", password, chromedp.ByID),
		chromedp.Click("btnLogin", chromedp.ByID))

	for !elementExists(ctx, "[data-testid='pay-with']") {
	}
	mustRunWithSuccessfulResp(ctx, chromedp.Click("payment-submit-btn", chromedp.ByID))

	if !isTest {
		for !elementExists(ctx, ".button__fast-track") {
		}
		mustRunWithSuccessfulResp(ctx, chromedp.Click(".button__fast-track", chromedp.ByQuery))
	}
}
