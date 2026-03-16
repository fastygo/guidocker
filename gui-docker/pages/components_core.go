package pages

import (
	"context"
	"github.com/a-h/templ"
	"html"
	"io"
	"strconv"
	"strings"
	kitlayout "ui8kit/layout"
)

func pageShell(data LayoutData, content templ.Component) templ.Component {
	return kitlayout.Shell(kitlayout.ShellProps{Title: data.Title, Active: data.Active, NavItems: NavigationItems()}, content)
}

func esc(value string) string {
	return html.EscapeString(value)
}

func write(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}

func renderStatusBadge(w io.Writer, status string) error {
	return write(w, `<span class="`+esc(StatusClass(status))+`">`+esc(StatusLabel(status))+`</span>`)
}

func renderContainerActions(w io.Writer, id, status string) error {
	button := func(className, action, label string) string {
		return `<button class="` + className + `" onclick="updateContainerStatus('' + esc(id) + '', '` + esc(action) + `')">` + esc(label) + `</button>`
	}
	switch status {
	case "running":
		return write(w, button("rounded border border-border bg-muted px-4 py-2 text-sm text-muted-foreground", "stop", "Stop")+button("rounded border border-border bg-muted px-4 py-2 text-sm text-muted-foreground", "pause", "Pause")+button("rounded border border-border bg-accent px-4 py-2 text-sm text-accent-foreground", "restart", "Restart"))
	case "stopped":
		return write(w, button("rounded border border-border bg-primary px-4 py-2 text-sm text-primary-foreground", "start", "Start"))
	case "paused":
		return write(w, button("rounded border border-border bg-accent px-4 py-2 text-sm text-accent-foreground", "unpause", "Unpause")+button("rounded border border-destructive bg-destructive px-4 py-2 text-sm text-destructive-foreground", "stop", "Stop"))
	default:
		return write(w, button("rounded border border-border bg-primary px-4 py-2 text-sm text-primary-foreground", "start", "Start"))
	}
}

func OverviewPage(data OverviewView) templ.Component {
	return pageShell(data.LayoutData, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := write(w, `<section class="space-y-4"><div class="text-sm text-muted-foreground">`+esc(data.Subtitle)+`</div><div class="text-2xl font-semibold">Overview</div><div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">`); err != nil {
			return err
		}
		cards := []struct {
			name  string
			value int
		}{{"Total Containers", data.Stats.TotalContainers}, {"Running", data.Stats.RunningContainers}, {"Stopped", data.Stats.StoppedContainers}, {"Paused", data.Stats.PausedContainers}}
		for _, card := range cards {
			if err := write(w, `<article class="card card-content"><div class="text-sm text-muted-foreground">`+esc(card.name)+`</div><p class="mt-2 text-3xl font-semibold">`+strconv.Itoa(card.value)+`</p></article>`); err != nil {
				return err
			}
		}
		if err := write(w, `</div><div class="card"><div class="card-header"><div class="card-title">Recent containers</div></div><div class="overflow-x-auto"><table class="w-full"><thead class="bg-muted"><tr><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Name</th><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Image</th><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Status</th><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Ports</th><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">CPU</th><th class="min-w-0 px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Memory</th><th class="px-4 py-2 text-left text-xs font-semibold text-muted-foreground">Actions</th></tr></thead><tbody class="divide-y divide-border">`); err != nil {
			return err
		}
		for _, c := range data.Containers {
			if err := write(w, `<tr><td class="min-w-0 px-4 py-2 text-sm font-medium"><a href="/apps/`+esc(c.ID)+`" class="block truncate">`+esc(c.Name)+`</a></td><td class="min-w-0 truncate px-4 py-2 text-sm text-muted-foreground">`+esc(c.Image)+`</td><td class="min-w-0 truncate px-4 py-2 text-sm">`); err != nil {
				return err
			}
			if err := renderStatusBadge(w, c.Status); err != nil {
				return err
			}
			if err := write(w, `</td><td class="min-w-0 truncate px-4 py-2 text-sm text-muted-foreground">`+esc(c.PortsStr)+`</td><td class="min-w-0 truncate px-4 py-2 text-sm text-muted-foreground">`+esc(c.CPUPercent)+`</td><td class="min-w-0 truncate px-4 py-2 text-sm text-muted-foreground">`+esc(c.MemoryMB)+`</td><td class="min-w-0 px-4 py-2 text-sm">`); err != nil {
				return err
			}
			if err := renderContainerActions(w, c.ID, c.Status); err != nil {
				return err
			}
			if err := write(w, `</td></tr>`); err != nil {
				return err
			}
		}
		return write(w, `</tbody></table></div></div></section>`)
	}))
}

func AppsPage(data AppsView) templ.Component {
	return pageShell(data.LayoutData, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := write(w, `<section class="space-y-4"><div class="flex items-center justify-between"><div><div class="text-sm text-muted-foreground">Managed containers</div><div class="text-2xl font-semibold">Apps</div></div><a href="/apps/new" class="rounded border border-border px-4 py-2 text-sm">Create app</a></div><div class="grid gap-2">`); err != nil {
			return err
		}
		if len(data.Items) == 0 {
			return write(w, `<div class="rounded border border-dashed border-border bg-card p-6 text-sm text-muted-foreground">No applications yet. Create your first stack from the Compose screen.</div></div></section>`)
		}
		for _, item := range data.Items {
			if err := write(w, `<a href="/apps/`+esc(item.ID)+`" class="app-row hover:bg-accent"><span class="truncate font-medium">`+esc(item.Name)+`</span>`); err != nil {
				return err
			}
			if err := renderStatusBadge(w, item.Status); err != nil {
				return err
			}
			if err := write(w, `</a>`); err != nil {
				return err
			}
		}
		return write(w, `</div></section>`)
	}))
}

func AppDetailPage(data AppDetailView) templ.Component {
	return pageShell(data.LayoutData, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := write(w, `<section class="space-y-4"><div><h2 class="text-2xl font-semibold">`+esc(data.Name)+`</h2><div class="text-sm text-muted-foreground">ID: `+esc(data.ID)+`</div></div><div class="grid gap-3 md:grid-cols-2"><div class="card card-content"><div class="font-semibold">Runtime</div><p class="text-sm text-muted-foreground">Image: `+esc(data.Image)+`</p><p class="text-sm text-muted-foreground">Ports: `+esc(data.PortsStr)+`</p><p class="text-sm text-muted-foreground">CPU: `+esc(data.CPUPercent)+` / Memory: `+esc(data.MemoryMB)+`</p><div class="mt-3">`); err != nil {
			return err
		}
		if err := renderStatusBadge(w, data.Status); err != nil {
			return err
		}
		if err := write(w, `</div></div><div class="card card-content"><div class="font-semibold">Actions</div><div class="mt-2">`); err != nil {
			return err
		}
		if err := renderContainerActions(w, data.ID, data.Status); err != nil {
			return err
		}
		return write(w, `</div><div class="mt-3 space-y-2"><a href="/apps/`+esc(data.ID)+`/compose" class="inline-block rounded border border-border px-4 py-2 text-sm">Compose</a><a href="/apps/`+esc(data.ID)+`/logs" class="inline-block rounded border border-border px-4 py-2 text-sm">Logs</a></div></div></div></section>`)
	}))
}

func ComposeContainerPage(data ComposeContainerView) templ.Component {
	return pageShell(data.LayoutData, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return write(w, `<section class="space-y-4"><div><div class="text-sm text-muted-foreground">`+esc(data.Subtitle)+`</div><div class="text-2xl font-semibold">`+esc(data.Title)+`</div></div><div class="grid gap-3 md:grid-cols-2"><div class="card card-content"><div class="font-semibold">Compose pack</div><form onsubmit="event.preventDefault(); showComposeMessage('Compose saved to local draft');" class="space-y-3"><div><label class="form-label">App name</label><input class="form-input" value="`+esc(data.Name)+`" /></div><div><label class="form-label">Service image</label><input class="form-input" value="`+esc(data.Image)+`" /></div><div><label class="form-label">Docker Compose yaml</label><textarea class="mt-1 w-full rounded border border-border bg-background p-2 text-sm" rows="14">`+esc(data.ComposeYAML)+`</textarea></div><div id="compose-status" class="text-sm text-muted-foreground"></div><div class="flex gap-2"><button type="submit" class="rounded border border-border px-4 py-2 text-sm">Save compose</button><button type="button" onclick="showComposeMessage('Compose deployment started')" class="rounded border border-border px-4 py-2 text-sm">Deploy now</button></div></form></div><div class="card card-content"><div class="font-semibold">Shortcuts</div><div class="mt-3 space-y-2 text-sm text-muted-foreground"><p>Use this screen as a blueprint for full compose-driven deployment.</p><p>You can paste env vars, volumes, networks and secrets manually into yaml.</p></div></div></div></section>`)
	}))
}

func LogsContainerPage(data LogsView) templ.Component {
	return pageShell(data.LayoutData, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return write(w, `<section class="space-y-4"><div><div class="text-sm text-muted-foreground">`+esc(data.ID)+`</div><div class="text-2xl font-semibold">`+esc(data.Name)+` logs</div><div class="text-sm text-muted-foreground">Recent runtime log output</div></div><div id="app-feedback" class="text-sm text-muted-foreground"></div><div class="flex gap-2"><button onclick="showComposeMessage('Logs refreshed')" class="rounded border border-border px-4 py-2 text-sm">Refresh</button><a href="/apps/`+esc(data.ID)+`/compose" class="rounded border border-border px-4 py-2 text-sm">Open compose</a></div><div class="card card-content overflow-x-auto"><pre class="whitespace-pre-wrap text-xs text-muted-foreground">`+esc(data.LogsContent)+`</pre></div></section>`)
	}))
}

func LoginPage(data LoginView) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		messageBlock := ""
		if strings.TrimSpace(data.Message) != "" {
			messageBlock = `<div class="mt-4 rounded border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">` + esc(data.Message) + `</div>`
		}
		return write(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width, initial-scale=1.0"/><title>PaaS Login</title><link rel="stylesheet" href="/static/css/app.css"/></head><body class="bg-background text-foreground"><div class="flex min-h-screen items-center justify-center p-6"><form method="post" action="/login" class="w-full max-w-[420px] rounded border border-border bg-card p-6 shadow-sm"><div><div class="text-sm text-muted-foreground">PaaS Console</div><h1 class="mt-2 text-3xl font-semibold">Sign in</h1></div>`+messageBlock+`<div class="mt-4 space-y-4"><label class="block text-sm">Username<input type="text" name="username" autocomplete="username" required class="form-input"/></label><label class="block text-sm">Password<input type="password" name="password" autocomplete="current-password" required class="form-input"/></label><button type="submit" class="rounded border border-border bg-primary px-4 py-2 text-sm text-primary-foreground">Login</button></div></form></div></body></html>`)
	})
}
