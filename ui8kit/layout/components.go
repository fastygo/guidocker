package layout

import (
	"context"
	"fmt"
	"github.com/a-h/templ"
	"html"
	"io"
	"strings"
	"ui8kit/utils"
)

type NavItem struct {
	Path  string
	Label string
	Icon  string
}

type SidebarProps struct {
	Items  []NavItem
	Active string
	Mobile bool
}

type HeaderProps struct {
	Title string
}

type ShellProps struct {
	Title    string
	Active   string
	NavItems []NavItem
}

const shellThemeScript = `(function () {
  const root = document.documentElement;
  const themeIcon = () => document.getElementById('theme-toggle-icon');
  const savedTheme = localStorage.getItem('dashboard-theme');
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const shouldUseDark = savedTheme === 'dark' || (!savedTheme && prefersDark);
  const applyThemeIcon = () => {
    const icon = themeIcon();
    if (!icon) return;
    icon.className = root.classList.contains('dark') ? 'latty latty-sun h-4 w-4' : 'latty latty-moon h-4 w-4';
  };
  root.classList.toggle('dark', shouldUseDark);
  applyThemeIcon();
  window.toggleDashboardTheme = function () {
    const nextDark = !root.classList.contains('dark');
    root.classList.toggle('dark', nextDark);
    localStorage.setItem('dashboard-theme', nextDark ? 'dark' : 'light');
    applyThemeIcon();
  };
  document.addEventListener('DOMContentLoaded', applyThemeIcon);
})();`

const shellPageScript = `window.openMobileSidebar = function () {
  const sidebar = document.getElementById('mobile-sidebar');
  const backdrop = document.getElementById('mobile-sidebar-backdrop');
  if (!sidebar || !backdrop) return;
  sidebar.classList.remove('hidden');
  backdrop.classList.remove('hidden');
  document.body.style.overflow = 'hidden';
};
window.closeMobileSidebar = function () {
  const sidebar = document.getElementById('mobile-sidebar');
  const backdrop = document.getElementById('mobile-sidebar-backdrop');
  if (!sidebar || !backdrop) return;
  sidebar.classList.add('hidden');
  backdrop.classList.add('hidden');
  document.body.style.overflow = '';
};
window.addEventListener('keydown', function (event) {
  if (event && event.key === 'Escape') closeMobileSidebar();
});
window.showComposeMessage = function (message) {
  const composeTarget = document.getElementById('compose-status');
  if (composeTarget) { composeTarget.textContent = message; return; }
  const appFeedback = document.getElementById('app-feedback');
  if (appFeedback) appFeedback.textContent = message;
};
window.updateContainerStatus = async function (containerID, status) {
  try {
    const response = await fetch('/api/containers/' + encodeURIComponent(containerID), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: status }),
    });
    const data = await response.json();
    if (data && data.success) location.reload();
  } catch (err) {
    console.error('Action failed:', err);
  }
};`

func writeString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}

func sidebarLinkClass(active, path string) string {
	if active == path {
		return "bg-accent text-accent-foreground"
	}
	return "text-muted-foreground hover:bg-accent"
}

func Sidebar(props SidebarProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		for _, item := range props.Items {
			onClick := ""
			if props.Mobile {
				onClick = ` onclick="closeMobileSidebar()"`
			}
			className := utils.Cn("flex items-center gap-2 rounded px-4 py-2 text-sm", sidebarLinkClass(props.Active, item.Path))
			if err := writeString(w, fmt.Sprintf(`<a href="%s" class="%s"%s><span class="latty latty-%s h-4 w-4"></span><span>%s</span></a>`, html.EscapeString(item.Path), html.EscapeString(className), onClick, html.EscapeString(item.Icon), html.EscapeString(item.Label))); err != nil {
				return err
			}
		}
		return nil
	})
}

func Header(props HeaderProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		return writeString(w, `<header class="h-15 flex items-center justify-between gap-2 border-b border-border px-4"><button onclick="openMobileSidebar()" class="inline-flex h-8 w-8 items-center justify-center rounded border border-border md:hidden" aria-label="Open menu"><span class="latty latty-menu h-4 w-4"></span></button><h1 class="flex-1 truncate px-2 text-base font-bold">`+html.EscapeString(props.Title)+`</h1><button onclick="toggleDashboardTheme()" class="inline-flex h-8 w-8 items-center justify-center rounded border border-border" aria-label="Toggle theme"><span id="theme-toggle-icon" class="latty latty-moon h-4 w-4"></span></button></header>`)
	})
}

func Shell(props ShellProps, content templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if strings.TrimSpace(props.Title) == "" {
			props.Title = "PaaS Dashboard"
		}
		if err := writeString(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width, initial-scale=1.0"/><title>`+html.EscapeString(props.Title)+`</title><link rel="stylesheet" href="/static/css/app.css"/><link rel="icon" type="image/svg+xml" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA0NyA0MCIgZmlsbD0iIzBlYTVlOSI+DQogICAgPHBhdGggZD0iTTIzLjUgNi41QzE3LjUgNi41IDEzLjc1IDkuNSAxMi4yNSAxNS41QzE0LjUgMTIuNSAxNy4xMjUgMTEuMzc1IDIwLjEyNSAxMi4xMjVDMjEuODM2NyAxMi41NTI5IDIzLjA2MDEgMTMuNzk0NyAyNC40MTQyIDE1LjE2OTJDMjYuNjIwMiAxNy40MDg0IDI5LjE3MzQgMjAgMzQuNzUgMjBDNDAuNzUgMjAgNDQuNSAxNyA0NiAxMUM0My43NSAxNCA0MS4xMjUgMTUuMTI1IDM4LjEyNSAxNC4zNzVDMzYuNDEzMyAxMy45NDcxIDM1LjE4OTkgMTIuNzA1MyAzMy44MzU3IDExLjMzMDhDMzEuNjI5NyA5LjA5MTU4IDI5LjA3NjYgNi41IDIzLjUgNi41Wk0xMi4yNSAyMEM2LjI1IDIwIDIuNSAyMyAxIDI5QzMuMjUgMjYgNS44NzUgMjQuODc1IDguODc1IDI1LjYyNUMxMC41ODY3IDI2LjA1MjkgMTEuODEwMSAyNy4yOTQ3IDEzLjE2NDIgMjguNjY5M0MxNS4zNzAyIDMwLjkwODQgMTcuOTIzNCAzMy41IDIzLjUgMzMuNUMyOS41IDMzLjUgMzMuMjUgMzAuNSAzNC43NSAyNC41QzMyLjUgMjcuNSAyOS44NzUgMjguNjI1IDI2Ljg3NSAyNy44NzVDMjUuMTYzMyAyNy40NDcxIDIzLjkzOTkgMjYuMjA1MyAyMi41ODU4IDI0LjgzMDdDMjAuMzc5OCAyMi41OTE2IDE3LjgyNjYgMjAgMTIuMjUgMjBaIj48L3BhdGg+DQo8L3N2Zz4="/><script>`+shellThemeScript+`</script></head><body class="min-h-screen overflow-x-hidden bg-background font-sans text-foreground"><div id="mobile-sidebar-backdrop" class="fixed inset-0 z-30 hidden" onclick="closeMobileSidebar()" style="background: rgba(0, 0, 0, 0.45);"></div><aside id="mobile-sidebar" class="fixed bottom-0 left-0 top-0 z-40 hidden w-64 max-w-full border-r border-border bg-card md:hidden"><div class="flex h-15 items-center justify-between border-b border-border px-4"><div><p class="font-bold">Local Panel</p></div><button onclick="closeMobileSidebar()" class="rounded border border-border px-2 py-1 text-xs">Close</button></div><nav class="space-y-1 p-2">`); err != nil {
			return err
		}
		if err := Sidebar(SidebarProps{Items: props.NavItems, Active: props.Active, Mobile: true}).Render(ctx, w); err != nil {
			return err
		}
		if err := writeString(w, `</nav></aside><div class="flex min-h-screen w-full"><aside class="hidden border-r border-border bg-card md:block md:w-64"><div class="flex h-15 items-center border-b border-border px-4"><p class="font-bold">Local Panel</p></div><nav class="space-y-1 p-2">`); err != nil {
			return err
		}
		if err := Sidebar(SidebarProps{Items: props.NavItems, Active: props.Active}).Render(ctx, w); err != nil {
			return err
		}
		if err := writeString(w, `</nav></aside><div class="flex min-w-0 flex-1 flex-col">`); err != nil {
			return err
		}
		if err := Header(HeaderProps{Title: props.Title}).Render(ctx, w); err != nil {
			return err
		}
		if err := writeString(w, `<main class="w-full min-w-0 max-w-full p-4 md:p-6">`); err != nil {
			return err
		}
		if content != nil {
			if err := content.Render(ctx, w); err != nil {
				return err
			}
		}
		return writeString(w, `</main></div></div><script>`+shellPageScript+`</script></body></html>`)
	})
}
