package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	botlog "github.com/partyfamine/bbbot/log"
	"github.com/partyfamine/bbbot/status"
)

type Bot struct {
	Sku             string
	Limit           float32
	RemainingFunds  *float32
	BestbuyEmail    string
	BestbuyPassword string
	PaylpalEmail    string
	PaylpalPassword string
	IsTest          bool
	Headless        bool
	log             botlog.Logger
	statusChecker   status.Checker
	mu              sync.Mutex
}

func (b *Bot) Exec(ctx context.Context) {
	b.log = botlog.New(b.Sku)
	defer func() {
		if err := b.log.Close(); err != nil {
			log.Fatal("error closing log", err)
		}
	}()

	botOpts := opts
	if b.Headless {
		botOpts = append(botOpts, chromedp.Headless, UserAgent)
	}

	for {
		if err := b.exec(ctx, botOpts); err != nil {
			if err == status.ErrStalled {
				b.log.Println("stalled; restarting")
				continue
			}
			log.Fatal("err running bot: ", err)
			break
		}
		break
	}
}

func (b *Bot) exec(parent context.Context, opts []chromedp.ExecAllocatorOption) error {
	allocateCtx, cancel := chromedp.NewExecAllocator(parent, opts...)
	ctx, _ := chromedp.NewContext(allocateCtx) //TODO: may not need
	checker := status.NewChecker(allocateCtx, cancel, time.Minute)
	b.statusChecker = checker
	defer checker.Close()

	err := b.runInOrder(
		b.navigateToPage,
		b.loopWhile(b.not(b.isInStockAndAddedToCart)))(ctx)
	if err != nil {
		return err
	}

	withinPriceRange, err := b.isWithinPriceRange(".priceView-customer-price span")(ctx)
	if err != nil {
		return err
	}
	if !withinPriceRange {
		b.log.Printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", *b.RemainingFunds)
		return nil
	}

	err = b.login(ctx)
	if err != nil {
		return err
	}

	withinPriceRange, err = b.isWithinPriceRange(".price-summary__total-value")(ctx)
	if err != nil {
		return err
	}
	if !withinPriceRange {
		b.log.Printf("exceeded price range, shutting down bot; remainingFunds: %.2f\n", *b.RemainingFunds)
		return nil
	}

	return b.payWithPaypal(ctx)
}

func (b *Bot) navigateToPage(ctx context.Context) error {
	url := fmt.Sprintf("http://www.bestbuy.com/site/%[1]s.p?skuId=%[1]s", b.Sku)
	title := ""

	b.log.Printf("navigating to %s\n", url)
	err := b.run(
		chromedp.Navigate(url),
		chromedp.Title(&title))(ctx)
	if err != nil {
		return err
	}

	b.log.Println(title)
	return nil
}

func (b *Bot) isInStock(ctx context.Context) (bool, error) {
	log.Println("checking stock")
	var btnText string
	addToCartSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-primary", b.Sku)
	disabledBtnSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-disabled", b.Sku)

	err := b.runInOrder(
		b.requireSuccess(chromedp.Reload()),
		b.loopWhile(
			b.not(
				b.or(
					b.elementExists(addToCartSelector),
					b.elementExists(disabledBtnSelector)))),
		b.ifTrue(
			b.elementExists(addToCartSelector),
			b.run(chromedp.Text(addToCartSelector, &btnText, chromedp.ByQuery)),
			b.run(chromedp.Text(disabledBtnSelector, &btnText, chromedp.ByQuery)),
		))(ctx)
	if err != nil {
		return false, err
	}

	if btnText == "Add to Cart" {
		b.log.Println("in stock!!!")
		return true, nil
	}
	b.log.Printf("out of stock: %s\n", btnText)
	return false, nil
}

func (b *Bot) addedToCart(ctx context.Context) (bool, error) {
	b.log.Println("adding to cart")
	addToCartSelector := fmt.Sprintf("[data-sku-id='%s'].add-to-cart-button.btn-primary", b.Sku)

	addToCartExists, err := b.elementExists(addToCartSelector)(ctx)
	if err != nil {
		return false, err
	}
	if !addToCartExists {
		b.log.Println("lol nevermind, can't add to cart; reloading")
		return false, nil
	}

	err = b.runInOrder(
		b.run(chromedp.Click(addToCartSelector, chromedp.ByQuery)),
		b.loopWhile(
			b.not(
				b.or(
					b.and(b.elementExists(".shop-alert"), b.not(b.elementEmpty(".shop-alert"))),
					b.elementExists(".shop-cart-icon .dot")))))(ctx)
	if err != nil {
		return false, err
	}

	alertExists, err := b.and(b.elementExists(".shop-alert"), b.not(b.elementEmpty(".shop-alert")))(ctx)
	if err != nil {
		return false, err
	}
	if alertExists {
		waitScriptExists, err := b.elementExists("style[class]")(ctx)
		if err != nil {
			return false, err
		}
		if !waitScriptExists {
			b.log.Println("alert encountered")
			return false, nil
		}
		b.log.Println("in queue, waiting...")
		err = b.runInOrder(
			b.loopWhile(b.elementExists("style[class]")),
			b.run(chromedp.Click(addToCartSelector, chromedp.ByQuery)),
			b.loopWhile(
				b.not(
					b.or(
						b.and(b.elementExists(".shop-alert"), b.not(b.elementEmpty(".shop-alert"))),
						b.elementExists(".shop-cart-icon .dot")))))(ctx)
		if err != nil {
			return false, err
		}
	}

	b.log.Println("no alerts")
	err = b.ifTrue(
		b.elementExists(".go-to-cart-button"),
		b.run(chromedp.Click(".go-to-cart-button", chromedp.ByQuery)),
		b.run(chromedp.Click(".cart-link", chromedp.ByQuery)),
	)(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bot) isInStockAndAddedToCart(ctx context.Context) (bool, error) {
	inStock, err := b.isInStock(ctx)
	if err != nil {
		return false, err
	}
	if !inStock {
		return false, nil
	}

	addedToCart, err := b.addedToCart(ctx)
	if err != nil {
		return false, err
	}
	if !addedToCart {
		b.log.Println("false alarm, nothing added")
		return false, nil
	}
	b.log.Println("moving to login")
	return true, nil
}

func (b *Bot) isWithinPriceRange(priceSelector string) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		if b.Limit == 0 {
			return true, nil
		}

		var priceStr string
		for priceStr == "" {
			err := b.runInOrder(
				b.loopWhile(
					b.not(b.elementExists(priceSelector))),
				b.run(chromedp.Text(priceSelector, &priceStr, chromedp.ByQuery)),
			)(ctx)
			if err != nil {
				return false, err
			}
		}

		b.mu.Lock()
		curFunds := *b.RemainingFunds
		b.mu.Unlock()

		gpuPrice, err := strToPrice(priceStr)
		if err != nil {
			return false, err
		}
		return float32(gpuPrice) < curFunds, nil
	}
}

func strToPrice(str string) (float32, error) {
	if str[0] == '$' {
		str = str[1:]
	}

	str = strings.ReplaceAll(str, ",", "")

	price, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return 0, err
	}

	return float32(price), nil
}

func (b *Bot) login(ctx context.Context) error {
	b.log.Println("logging in")

	err := b.runInOrder(
		b.loopWhile(
			b.not(b.elementExists(".price-summary__total-value"))),
		b.run(chromedp.Click(".account-button", chromedp.ByQuery)),
		b.loopWhile(
			b.not(b.elementExists(".account-menu .sign-in-btn"))),
		b.requireSuccess(chromedp.Click(".account-menu .sign-in-btn", chromedp.ByQuery)),
		b.loopWhile(
			b.not(b.elementExists("#fld-e"))),
		b.run(chromedp.SendKeys("fld-e", b.BestbuyEmail, chromedp.ByID)),
		b.run(chromedp.SendKeys("fld-p1", b.BestbuyPassword, chromedp.ByID)),
		b.requireSuccess(chromedp.Click(".cia-form__controls__submit", chromedp.ByQuery)),
	)(ctx)
	if err != nil {
		return err
	}

	b.log.Println("signed in")
	return nil
}

func (b *Bot) payWithPaypal(ctx context.Context) error {
	b.log.Println("checking out via paypal")

	var priceStr string
	err := b.run(chromedp.Text(".price-summary__total-value", &priceStr, chromedp.ByQuery))(ctx)
	if err != nil {
		return err
	}

	if b.Limit != 0 {
		cartPrice, err := strToPrice(priceStr)
		if err != nil {
			return err
		}

		b.mu.Lock()
		*b.RemainingFunds -= float32(cartPrice)
		b.mu.Unlock()
	}

	correctItemAdded, err := b.elementExists("#cart-" + b.Sku)(ctx)
	if err != nil {
		return err
	}
	if !correctItemAdded {
		return errors.New("wrong item added to cart, aborting")
	}

	b.log.Println("correct item added")
	err = b.runInOrder(
		b.loopWhile(
			b.not(b.elementExists(".checkout-buttons .checkout-buttons__paypal"))),
		b.run(chromedp.Sleep(1*time.Second)),
		b.requireSuccess(chromedp.Click(".checkout-buttons .checkout-buttons__paypal", chromedp.ByQuery)),
		b.run(
			chromedp.SendKeys("email", b.PaylpalEmail, chromedp.ByID),
			chromedp.Click("btnNext", chromedp.ByID)),
		b.run(
			chromedp.SendKeys("password", b.PaylpalPassword, chromedp.ByID),
			chromedp.Click("btnLogin", chromedp.ByID)),
		b.loopWhile(
			b.not(b.elementExists("[data-testid='pay-with']"))),
		b.run(chromedp.Sleep(2*time.Second)),
		b.requireSuccess(chromedp.Click("payment-submit-btn", chromedp.ByID)),
	)(ctx)
	if err != nil {
		return err
	}

	if b.IsTest {
		b.log.Println("is test!!!")
		return nil
	}

	b.log.Println("not a test!!!")

	// err = b.runInOrder(
	// 	b.loopWhile(
	// 		b.not(b.elementExists(".button__fast-track"))),
	// 	b.requireSuccess(chromedp.Click(".button__fast-track", chromedp.ByQuery)),
	// )(ctx)
	// if err != nil {
	// 	return err
	// }

	time.Sleep(30 * time.Second) //extra time to finish processing just in case
	return nil
}
