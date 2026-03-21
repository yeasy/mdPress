package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/yeasy/mdpress/internal/pdf"
	"github.com/yeasy/mdpress/pkg/utils"
)

type mermaidRenderStatus struct {
	Done      bool   `json:"done"`
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	Total     int    `json:"total"`
	Rendered  int    `json:"rendered"`
	Processed int    `json:"processed"`
}

func validateRenderedMermaidHTML(htmlContent string) error {
	if htmlContent == "" {
		return nil
	}

	if err := pdf.CheckChromiumAvailable(); err != nil {
		return fmt.Errorf("real Mermaid render check unavailable: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "mdpress-mermaid-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temporary Mermaid validation file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	fullHTML := buildMermaidValidationHTML(htmlContent)
	if _, err := tmpFile.WriteString(fullHTML); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("failed to write temporary Mermaid validation file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary Mermaid validation file: %w", err)
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fileURL := "file://" + tmpPath
	var status mermaidRenderStatus
	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				if err := chromedp.Run(ctx, chromedp.Evaluate(`window.__mdpressMermaidStatus`, &status)); err != nil {
					return err
				}
				if status.Done {
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
			if !status.Done {
				return fmt.Errorf("mermaid rendering timed out")
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to execute Mermaid render check: %w", err)
	}

	if !status.OK {
		if status.Error == "" {
			status.Error = "Mermaid did not render successfully"
		}
		return fmt.Errorf("%s", status.Error)
	}
	if status.Total > 0 && status.Processed < status.Total {
		return fmt.Errorf("only %d/%d Mermaid block(s) rendered", status.Processed, status.Total)
	}

	return nil
}

func buildMermaidValidationHTML(bodyHTML string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>mdpress Mermaid Validation</title>
</head>
<body>
` + bodyHTML + `
<script>
window.__mdpressMermaidStatus = { done: false, ok: false, error: "", total: 0, rendered: 0, processed: 0 };
window.addEventListener('error', function(event) {
  if (!window.__mdpressMermaidStatus.done) {
    window.__mdpressMermaidStatus.error = event.message || String(event.error || event);
    window.__mdpressMermaidStatus.done = true;
  }
});
</script>
<script src="` + utils.MermaidCDNURL + `"></script>
<script>
(async function() {
  try {
    var nodes = Array.from(document.querySelectorAll('.mermaid'));
    window.__mdpressMermaidStatus.total = nodes.length;
    if (nodes.length === 0) {
      window.__mdpressMermaidStatus.ok = true;
      return;
    }
    if (!window.mermaid) {
      throw new Error('Mermaid library failed to load');
    }
    mermaid.initialize({ startOnLoad: false, theme: 'default', securityLevel: 'loose', themeVariables: { fontFamily: '"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif' } });
    await mermaid.run({ querySelector: '.mermaid' });
    var processed = 0;
    nodes.forEach(function(node) {
      if (node.querySelector('svg') || node.getAttribute('data-processed') === 'true') {
        processed += 1;
      }
    });
    window.__mdpressMermaidStatus.rendered = document.querySelectorAll('.mermaid svg').length;
    window.__mdpressMermaidStatus.processed = processed;
    if (processed !== nodes.length) {
      throw new Error('Not all Mermaid diagrams produced rendered output');
    }
    window.__mdpressMermaidStatus.ok = true;
  } catch (error) {
    window.__mdpressMermaidStatus.error = String(error && error.message || error);
  } finally {
    window.__mdpressMermaidStatus.done = true;
  }
})();
</script>
</body>
</html>`
}
