package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

type bot struct {
	sku          string
	url          string
	logFile      *os.File
	statusTicker int64
}

func newBot(sku string) *bot {
	return &bot{
		sku: sku,
		url: fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", sku),
	}
}

func (b *bot) execBot(parentCtx context.Context) {
	f, err := os.Create(fmt.Sprintf("bot-%s.log", b.sku))
	if err != nil {

		if err == context.Canceled {
			fmt.Println("is context cancelled error")
		}
		log.Fatalf("%T, %v\n", err, err)
	}
	defer f.Close()
	b.logFile = f

	b.url = fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", b.sku)

	for {
		botOpts := opts
		if headless {
			botOpts = append(botOpts, chromedp.Headless, UserAgent)
		}
		allocateCtx, cancelAllocate := chromedp.NewExecAllocator(parentCtx, botOpts...)
		ctx, cancel := chromedp.NewContext(allocateCtx)

		go b.monitor(ctx, cancel)
		success := b.exec(ctx)
		if success {
			time.Sleep(30 * time.Second) //extra time to finish processing just in case
			cancelAllocate()
			cancel()
			break
		}
		cancelAllocate()
		cancel()
	}
}

func (b *bot) exec(ctx context.Context) bool {
	b.navigateToPage(ctx)

	for {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		inStock := b.isInStock(ctx)
		if inStock {
			added := b.addToCart(ctx)
			if !added {
				b.println("false alarm, nothing added")
				continue
			}
			if inStock {
				b.println("moving to login")
				break
			}
		}
	}

	select {
	case <-ctx.Done():
		return false
	default:
	}

	if !b.withinPriceRange(ctx, ".priceView-customer-price span") {
		b.printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", remainingFunds)
		return true
	}
	b.login(ctx)
	if !b.withinPriceRange(ctx, ".price-summary__total-value") {
		b.printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", remainingFunds)
		return true
	}
	b.payWithPaypal(ctx)

	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

func (b *bot) monitor(ctx context.Context, cancel context.CancelFunc) {
	lastStatus := 0
	for {
		time.Sleep(1 * time.Minute)
		mu.Lock()
		curStatus := int(b.statusTicker)
		mu.Unlock()

		if lastStatus == curStatus {
			break
		}

		lastStatus = curStatus

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	b.println("stalled; cancelling and reloading")
	cancel()
}

func (b *bot) navigateToPage(ctx context.Context) {
	title := ""
	b.printf("navigating to %s\n", b.url)
	err := chromedp.Run(ctx,
		chromedp.Navigate(b.url),
		chromedp.Title(&title))
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
	b.println(title)
}

func (b *bot) isInStock(ctx context.Context) bool {
	var btnText string

	b.mustRunWithSuccessfulResp(ctx, chromedp.Reload())

	addToCartSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-primary", b.sku)
	disabledBtnSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-disabled", b.sku)
	for {
		addToCartExists, err := elementExists(ctx, addToCartSelector)
		if err != nil {
			return false
		}

		disabledBtnExists, err := elementExists(ctx, disabledBtnSelector)
		if err != nil {
			return false
		}

		if addToCartExists || disabledBtnExists {
			break
		}
	}

	btnForText := addToCartSelector
	disabledBtnExists, err := elementExists(ctx, disabledBtnSelector)
	if err != nil {
		return false
	}

	if disabledBtnExists {
		btnForText = disabledBtnSelector
	}

	b.mustRun(ctx, chromedp.Text(btnForText, &btnText, chromedp.ByQuery))

	if btnText == "Add to Cart" {
		b.println("in stock!!!")
		return true
	}
	b.printf("out of stock: %s\n", btnText)
	return false
}

func (b *bot) addToCart(ctx context.Context) bool {
	b.println("adding to cart")
	addToCartSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-primary", b.sku)
	if exists, err := elementExists(ctx, addToCartSelector); !exists || err != nil {
		b.println("lol nevermind, can't add to cart; reloading")
		return false
	}
	b.mustRun(ctx, chromedp.Click(addToCartSelector, chromedp.ByQuery))

	for {
		alertExists, err := elementExists(ctx, ".c-alert-content")
		if err != nil {
			return false
		}

		itemsInCart, err := elementExists(ctx, ".shop-cart-icon .dot")
		if err != nil {
			return false
		}

		if alertExists || itemsInCart {
			break
		}
	}

	if exists, err := elementExists(ctx, ".c-alert-content"); exists || err != nil {
		b.println("alert encountered")
		return false
	}

	goToCart := ".go-to-cart-button"
	cartExists, err := elementExists(ctx, goToCart)
	if err != nil {
		return false
	}
	if !cartExists {
		goToCart = ".cart-link"
	}
	b.println("added to cart")
	b.mustRun(ctx, chromedp.Click(goToCart, chromedp.ByQuery))
	b.println("loaded cart")

	return true
}

func (b *bot) payWithPaypal(ctx context.Context) {
	b.println("checking out via paypal")
	priceStr := ""
	b.mustRun(ctx, chromedp.Text(".price-summary__total-value", &priceStr, chromedp.ByQuery))

	if limit != 0 {
		cartPrice := mustStrToPrice(priceStr)
		mu.Lock()
		remainingFunds -= float32(cartPrice)
		mu.Unlock()
	}

	if exists, err := elementExists(ctx, "#cart-"+b.sku); !exists || err != nil {
		log.Fatal("wrong item added to cart, aborting")
	}

	b.mustRunWithSuccessfulResp(ctx, chromedp.Click(".checkout-buttons__paypal", chromedp.ByQuery))

	b.mustRun(ctx,
		chromedp.SendKeys("email", paylpalEmail, chromedp.ByID),
		chromedp.Click("btnNext", chromedp.ByID))

	b.mustRun(ctx,
		chromedp.SendKeys("password", paylpalPassword, chromedp.ByID),
		chromedp.Click("btnLogin", chromedp.ByID))

	for {
		exists, err := elementExists(ctx, "[data-testid='pay-with']")
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		if exists {
			break
		}
	}
	time.Sleep(2 * time.Second) //needs a lil bit to fully load page
	b.mustRunWithSuccessfulResp(ctx, chromedp.Click("payment-submit-btn", chromedp.ByID))

	if !isTest {
		for {
			exists, err := elementExists(ctx, ".button__fast-track")
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Fatal(err)
			}
			if exists {
				break
			}
		}
		b.mustRunWithSuccessfulResp(ctx, chromedp.Click(".button__fast-track", chromedp.ByQuery))
	}
}

func (b *bot) login(ctx context.Context) {
	for {
		exists, err := elementExists(ctx, ".price-summary__total-value")
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		if exists {
			break
		}
	}
	b.mustRun(ctx, chromedp.Click(".account-button", chromedp.ByQuery))
	b.println("clicking account button")
	for {
		exists, err := elementExists(ctx, "#ABT2465Menu .lam-signIn__button")
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		if exists {
			break
		}
	}
	b.mustRunWithSuccessfulResp(ctx, chromedp.Click("#ABT2465Menu .lam-signIn__button", chromedp.ByQuery))
	b.println("signing in")
	for {
		exists, err := elementExists(ctx, "#fld-e")
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		if exists {
			break
		}
	}
	b.mustRun(ctx, chromedp.SendKeys("fld-e", bestbuyEmail, chromedp.ByID))
	b.mustRun(ctx, chromedp.SendKeys("fld-p1", bestbuyPassword, chromedp.ByID))
	b.mustRunWithSuccessfulResp(ctx, chromedp.Click(".cia-form__controls__submit", chromedp.ByQuery))
	b.println("signed in")
}

func (b *bot) declineSurvey(ctx context.Context) {
	exists, err := elementExists(ctx, "#survey_invite_no")
	if err != nil {
		if err == context.Canceled {
			return
		}
		log.Fatal(err)
	}
	if exists {
		b.println("declining survey")
		err := chromedp.Run(ctx, chromedp.Click("#survey_invite_no", chromedp.ByID))
		if err != nil && err != context.Canceled {
			log.Fatal(err)
		}
	}
}

func (b *bot) println(msg string) {
	fmt.Fprintf(b.logFile, "%s: %s\n", time.Now().Format(time.Stamp), msg)
	log.Printf("%s: %s\n", b.sku, msg)
}

func (b *bot) printf(msg string, params ...interface{}) {
	fmtMsg := fmt.Sprintf(msg, params...)
	fmt.Fprintf(b.logFile, "%s: %s", time.Now().Format(time.Stamp), fmtMsg)
	log.Printf("%s: %s", b.sku, fmtMsg)
}

func (b *bot) mustRun(ctx context.Context, actions ...chromedp.Action) {
	b.declineSurvey(ctx)
	err := chromedp.Run(ctx, actions...)
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
	b.statusTicker++
}

func (b *bot) mustRunWithSuccessfulResp(ctx context.Context, actions ...chromedp.Action) {
	for {
		b.declineSurvey(ctx)
		resp, err := chromedp.RunResponse(ctx, actions...)
		if err != nil && err != context.Canceled {
			log.Fatal(err)
		}

		if resp != nil {
			break
		}
	}
	b.statusTicker++
}

func (b *bot) withinPriceRange(ctx context.Context, priceSelector string) bool {
	if limit == 0 {
		return true
	}

	var priceStr string
	b.declineSurvey(ctx)
	err := chromedp.Run(ctx, chromedp.Text(priceSelector, &priceStr, chromedp.ByQuery))
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}

	mu.Lock()
	curFunds := remainingFunds
	mu.Unlock()

	gpuPrice := mustStrToPrice(priceStr)
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
	return float32(gpuPrice) < curFunds
}
