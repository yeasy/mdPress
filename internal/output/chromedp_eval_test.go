package output

import (
	"context"

	"github.com/chromedp/chromedp"
)

// evalInto runs chromedp.Evaluate as a step of chromedp.Do and stores the
// decoded result in dst, so a test can mix evaluations with other steps.
func evalInto[T any](expr string, dst *T, opts ...chromedp.EvaluateOption) chromedp.Action[chromedp.Void] {
	return func(ctx context.Context, t *chromedp.Target) (chromedp.Void, error) {
		v, err := chromedp.Evaluate[T](expr, opts...)(ctx, t)
		if err == nil {
			*dst = v
		}
		return chromedp.Void{}, err
	}
}
