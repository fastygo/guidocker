package pages

import (
	"fmt"
	"gui-docker/domain"
	"strings"
	"time"
	kitlayout "ui8kit/layout"
	"ui8kit/utils"
)

func FormatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func JoinStrings(items []string, sep string) string {
	return strings.Join(items, sep)
}

func TimeLayout() string {
	return "2006-01-02 15:04:05"
}

func FormatTime(t time.Time) string {
	return t.Format(TimeLayout())
}

func StatusClass(status string) string {
	return utils.StatusBadgeClass(status)
}

func StatusLabel(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "running":
		return "Running"
	case "stopped":
		return "Stopped"
	case "paused":
		return "Paused"
	case "created":
		return "Created"
	case "deploying":
		return "Deploying"
	case "error":
		return "Error"
	default:
		return domain.FormatStatusLabel(status)
	}
}

func ContainerToView(c domain.Container) ContainerView {
	return ContainerView{
		ID:         c.ID,
		Name:       c.Name,
		Image:      c.Image,
		Status:     domain.NormalizeStoredStatus(c.Status),
		PortsStr:   strings.Join(c.Ports, ", "),
		CPUPercent: c.GetCPUUsagePercent(),
		MemoryMB:   c.GetMemoryUsageMB(),
	}
}

func ContainersToViews(containers []domain.Container) []ContainerView {
	out := make([]ContainerView, len(containers))
	for i, c := range containers {
		out[i] = ContainerToView(c)
	}
	return out
}

func NavigationItems() []kitlayout.NavItem {
	return []kitlayout.NavItem{
		{Path: "/", Label: "Overview", Icon: "house"},
		{Path: "/apps", Label: "Apps", Icon: "box"},
		{Path: "/scan", Label: "Scanner", Icon: "scan"},
		{Path: "/settings", Label: "Settings", Icon: "settings"},
	}
}
