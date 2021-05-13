package cmd

import (
	"context"
	"log"
	"time"

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
		mustRunWithSuccessfulResp(ctx, chromedp.Reload())

		for !elementExists(ctx, ".add-to-cart-button") {
		}
		mustRun(ctx, chromedp.Text(".add-to-cart-button", &btnText, chromedp.ByQuery))

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
	log.Println("added to cart")
	mustRunWithSuccessfulResp(ctx, chromedp.Click(goToCart, chromedp.ByQuery))
	log.Println("loaded cart")
}

func payWithPaypal(ctx context.Context) {
	log.Println("checking out via paypal")
	priceStr := ""
	mustRun(ctx, chromedp.Text(".price-summary__total-value", &priceStr, chromedp.ByQuery))

	if limit != 0 {
		cartPrice := mustStrToPrice(priceStr)
		mu.Lock()
		remainingFunds -= float32(cartPrice)
		mu.Unlock()
	}

	mustRunWithSuccessfulResp(ctx, chromedp.Click(".checkout-buttons__paypal", chromedp.ByQuery))

	mustRun(ctx,
		chromedp.SendKeys("email", paylpalEmail, chromedp.ByID),
		chromedp.Click("btnNext", chromedp.ByID))

	mustRun(ctx,
		chromedp.SendKeys("password", paylpalPassword, chromedp.ByID),
		chromedp.Click("btnLogin", chromedp.ByID))

	for !elementExists(ctx, "[data-testid='pay-with']") {
	}
	time.Sleep(2 * time.Second) //needs a lil bit to fully load page
	mustRunWithSuccessfulResp(ctx, chromedp.Click("payment-submit-btn", chromedp.ByID))

	if !isTest {
		for !elementExists(ctx, ".button__fast-track") {
		}
		mustRunWithSuccessfulResp(ctx, chromedp.Click(".button__fast-track", chromedp.ByQuery))
	}
}

func login(ctx context.Context) {
	for !elementExists(ctx, ".price-summary__total-value") {
	}
	mustRun(ctx, chromedp.Click(".account-button", chromedp.ByQuery))
	log.Println("clicking account button")
	for !elementExists(ctx, "#ABT2465Menu .lam-signIn__button") {
	}
	mustRunWithSuccessfulResp(ctx, chromedp.Click("#ABT2465Menu .lam-signIn__button", chromedp.ByQuery))
	log.Println("signing in")
	for !elementExists(ctx, "#fld-e") {
	}
	mustRun(ctx, chromedp.SendKeys("fld-e", bestbuyEmail, chromedp.ByID))
	mustRun(ctx, chromedp.SendKeys("fld-p1", bestbuyPassword, chromedp.ByID))
	mustRunWithSuccessfulResp(ctx, chromedp.Click(".cia-form__controls__submit", chromedp.ByQuery))
	log.Println("signed in")
}

func declineSurvey(ctx context.Context) {
	if elementExists(ctx, "#survey_invite_no") {
		err := chromedp.Run(ctx, chromedp.Click("#survey_invite_no", chromedp.ByID))
		if err != nil {
			log.Fatal(err)
		}
	}
}
