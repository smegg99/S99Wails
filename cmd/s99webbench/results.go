// cmd/s99webbench/results.go

//go:build linux

package main

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"charm.land/lipgloss/v2"

	"github.com/smegg99/s99wails/v2/internal/cli"
	"github.com/smegg99/s99wails/v2/internal/cli/catalog"
	"github.com/smegg99/s99wails/v2/webbench"
)

// result is one measurement as a titled two-column table.
func result(a *cli.App, r webbench.Result) string {
	l := a.Say()
	ms := func(v float64) string { return fmt.Sprintf("%.1f ms", v) }
	rows := [][2]string{
		{catalog.WebbenchRowLoadFinished(l), ms(r.LoadFinishedMS)},
		{catalog.WebbenchRowDomContent(l), ms(r.DOMContentMS)},
		{catalog.WebbenchRowLoadEvent(l), ms(r.LoadEventMS)},
		{catalog.WebbenchRowFirstPaint(l), ms(r.FirstPaintMS)},
		{catalog.WebbenchRowFirstContentfulPaint(l), ms(r.FCPMS)},
		{catalog.WebbenchRowFps(l), fmt.Sprintf("%.1f", r.FPS)},
		{catalog.WebbenchRowFrameP50(l), ms(r.FrameP50MS)},
		{catalog.WebbenchRowFrameP95(l), ms(r.FrameP95MS)},
		{catalog.WebbenchRowFrameWorst(l), ms(r.FrameWorstMS)},
		{catalog.WebbenchRowFramesCounted(l), fmt.Sprint(r.FramesCounted)},
		{catalog.WebbenchRowStyleRecalc(l), ms(r.StyleRecalcMS)},
		{catalog.WebbenchRowLayoutThrash(l), ms(r.LayoutThrashMS)},
		{catalog.WebbenchRowScript(l), ms(r.ScriptMS)},
		{catalog.WebbenchRowReactivity(l), ms(r.ReactivityMS)},
		{catalog.WebbenchRowNodes(l), fmt.Sprint(r.Nodes)},
		{catalog.WebbenchRowCssRules(l), fmt.Sprint(r.CSSRules)},
		{catalog.WebbenchRowStylesheets(l), fmt.Sprint(r.Stylesheets)},
	}
	for _, process := range slices.Sorted(maps.Keys(r.CPU)) {
		rows = append(rows, [2]string{
			catalog.WebbenchRowCpu(l, catalog.WebbenchRowCpuParams{Process: process}),
			fmt.Sprintf("%.1f%%", r.CPU[process]),
		})
	}
	styles := a.Console().Styles()
	table := styles.Table().Border(lipgloss.HiddenBorder()).BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).BorderColumn(false)
	for _, row := range rows {
		table = table.Row(row[0], row[1])
	}
	return lipgloss.JoinVertical(lipgloss.Left, styles.Title.Render(r.Label), table.Render()) + "\n"
}

// comparison renders results fastest first, each frame rate as a multiple of the slowest.
func comparison(a *cli.App, results []webbench.Result) string {
	l := a.Say()
	ordered := append([]webbench.Result(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].FPS > ordered[j].FPS })
	slowest := 0.0
	for _, r := range ordered {
		if r.FPS > 0 && (slowest == 0 || r.FPS < slowest) {
			slowest = r.FPS
		}
	}
	table := a.Console().Styles().Table().Headers(
		catalog.WebbenchColConfiguration(l), catalog.WebbenchColFps(l), catalog.WebbenchColP95(l),
		catalog.WebbenchColWorst(l), catalog.WebbenchColLoad(l), catalog.WebbenchColStyle(l),
		catalog.WebbenchColGpu(l), catalog.WebbenchColSpeed(l), catalog.WebbenchColError(l),
	)
	for _, r := range ordered {
		if r.Error != "" && r.FPS == 0 {
			table = table.Row(r.Label, "-", "-", "-", "-", "-", "-", "-", r.Error)
			continue
		}
		speed := "-"
		if slowest > 0 && r.FPS > 0 {
			speed = fmt.Sprintf("%.1fx", r.FPS/slowest)
		}
		table = table.Row(r.Label, fmt.Sprintf("%.1f", r.FPS), fmt.Sprintf("%.1f", r.FrameP95MS),
			fmt.Sprintf("%.1f", r.FrameWorstMS), fmt.Sprintf("%.0f", r.LoadFinishedMS),
			fmt.Sprintf("%.1f", r.StyleRecalcMS), fmt.Sprintf("%.1f", r.CPU["WebKitGPUProcess"]), speed, r.Error)
	}
	return table.Render() + "\n"
}
