package interfaces

import (
	"dashboard/domain"
	"dashboard/views"
	"net/http"
	"sort"
	"strings"
)

// HandleScan renders the scan page.
func (h *DashboardHandler) HandleScan(w http.ResponseWriter, r *http.Request) {
	if h.scanUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "Scanner is not configured")
		return
	}
	if r.Method != http.MethodGet {
		h.writeMethodNotAllowed(w)
		return
	}

	report, err := h.scanUseCase.RunScan(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	view := scanReportToView(report)
	h.executeView(w, "scan", view)
}

// APIScan returns scan report as JSON.
func (h *DashboardHandler) APIScan(w http.ResponseWriter, r *http.Request) {
	if h.scanUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "Scanner is not configured")
		return
	}
	if r.Method != http.MethodGet {
		h.writeMethodNotAllowed(w)
		return
	}

	report, err := h.scanUseCase.RunScan(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.writeJSON(w, http.StatusOK, report)
}

func scanReportToView(report *domain.ScanReport) views.ScanView {
	if report == nil {
		return views.ScanView{
			LayoutData: views.LayoutData{
				Title:    "PaaS Dashboard",
				Subtitle: "Scanner",
				Active:   "/scan",
			},
			ScannedAt: "unknown",
		}
	}

	resources := make([]views.ScanResourceView, 0, len(report.Resources))
	counts := views.ScanSummary{
		Managed:       0,
		Broken:        0,
		OrphanRuntime: 0,
		OrphanDir:     0,
		StaleAdmin:    0,
		Unknown:       0,
	}

	for _, resource := range report.Resources {
		switch resource.Kind {
		case domain.ResourceManaged:
			counts.Managed++
		case domain.ResourceBrokenApp:
			counts.Broken++
		case domain.ResourceOrphanRuntime:
			counts.OrphanRuntime++
		case domain.ResourceOrphanDir:
			counts.OrphanDir++
		case domain.ResourceStaleAdmin:
			counts.StaleAdmin++
		case domain.ResourceUnknown:
			counts.Unknown++
		}

		containerNames := append([]string(nil), resource.ContainerNames...)
		sort.Strings(containerNames)

		ports := append([]string(nil), resource.Ports...)
		sort.Strings(ports)

		resources = append(resources, views.ScanResourceView{
			Kind:        string(resource.Kind),
			Confidence:  string(resource.Confidence),
			Name:        resource.Name,
			Dir:         resource.Dir,
			Status:      resource.Status,
			Ports:       ports,
			ComposeProject: resource.ComposeProject,
			Reason:      resource.Reason,
			Containers:  containerNames,
			CleanupCmds: resource.CleanupCmds,
			IsCurrent:   resource.IsCurrentAdmin,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Kind == resources[j].Kind {
			return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
		}
		return resources[i].Kind < resources[j].Kind
	})

	return views.ScanView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Scanner",
			Active:   "/scan",
		},
		ScannedAt: views.FormatTime(report.ScannedAt),
		Resources: resources,
		StacksDir: report.StacksDir,
		Summary:   counts,
	}
}

