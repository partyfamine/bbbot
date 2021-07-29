package bot

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type step func(ctx context.Context) error
type conditionalStep func(ctx context.Context) (bool, error)

func (b *Bot) runInOrder(steps ...step) step {
	return func(ctx context.Context) error {
		for _, step := range steps {
			select {
			case <-ctx.Done():
				return b.statusChecker.Check()
			default:
			}
			b.statusChecker.Ping()
			if err := step(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func (b *Bot) loopWhile(condition conditionalStep) step {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return b.statusChecker.Check()
			default:
			}
			b.statusChecker.Ping()
			shouldContinue, err := condition(ctx)
			if err != nil {
				return err
			}
			if !shouldContinue {
				return nil
			}
		}
		return nil
	}
}

func (b *Bot) or(conditions ...conditionalStep) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		overallResult := false
		for _, condition := range conditions {
			select {
			case <-ctx.Done():
				return false, b.statusChecker.Check()
			default:
			}
			b.statusChecker.Ping()
			conditionResult, err := condition(ctx)
			if err != nil {
				return false, err
			}
			if conditionResult {
				overallResult = true
				break
			}
		}
		return overallResult, nil
	}
}

func (b *Bot) and(conditions ...conditionalStep) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		for _, condition := range conditions {
			select {
			case <-ctx.Done():
				return false, b.statusChecker.Check()
			default:
			}
			b.statusChecker.Ping()
			conditionResult, err := condition(ctx)
			if err != nil {
				return false, err
			}
			if !conditionResult {
				return false, nil
			}
		}
		return true, nil
	}
}

func (b *Bot) not(condition conditionalStep) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		select {
		case <-ctx.Done():
			return false, b.statusChecker.Check()
		default:
		}
		b.statusChecker.Ping()
		result, err := condition(ctx)
		if err != nil {
			return false, err
		}
		return !result, nil
	}
}

func (b *Bot) ifTrue(condition conditionalStep, then step, otherwise ...step) step {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return b.statusChecker.Check()
		default:
		}
		b.statusChecker.Ping()
		result, err := condition(ctx)
		if err != nil {
			return err
		}
		if result {
			return then(ctx)
		} else if otherwise != nil {
			return b.runInOrder(otherwise...)(ctx)
		}
		return nil
	}
}

func (b *Bot) elementExists(path string) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		select {
		case <-ctx.Done():
			return false, b.statusChecker.Check()
		default:
		}
		var nodes []*cdp.Node
		err := b.run(chromedp.Nodes(path, &nodes, chromedp.AtLeast(0), chromedp.ByQuery))(ctx)
		if err != nil {
			return false, err
		}
		return len(nodes) > 0, nil
	}
}

func (b *Bot) elementEmpty(path string) conditionalStep {
	return func(ctx context.Context) (bool, error) {
		select {
		case <-ctx.Done():
			return false, b.statusChecker.Check()
		default:
		}
		var html string
		err := b.run(chromedp.InnerHTML(path, &html, chromedp.AtLeast(0), chromedp.ByQuery))(ctx)
		if err != nil {
			return false, err
		}
		return html == "", nil
	}
}

func (b *Bot) run(actions ...chromedp.Action) step {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return b.statusChecker.Check()
		default:
		}
		b.statusChecker.Ping()
		err := chromedp.Run(ctx, actions...)
		if err != nil {
			return err
		}
		return nil
	}
}

func (b *Bot) requireSuccess(actions ...chromedp.Action) step {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return b.statusChecker.Check()
			default:
			}
			b.statusChecker.Ping()
			resp, err := chromedp.RunResponse(ctx, actions...)
			if err != nil {
				return err
			}
			if resp != nil {
				return nil
			}
		}
	}
}
