package components

import (
	"fmt"
	"strings"
)

// Weekday labels (Monday first)
var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Heat intensity colors for tview
var heatColors = []string{"gray", "blue", "green", "yellow", "red"}

// RenderHeatmap creates a colored text-based heatmap for hour/weekday commits
func RenderHeatmap(matrix [7][24]int, maxValue int) string {
	var sb strings.Builder

	// Header: hours
	sb.WriteString("      ")
	for h := 0; h < 24; h++ {
		if h%3 == 0 {
			sb.WriteString(fmt.Sprintf("[white]%02d[-] ", h))
		} else {
			sb.WriteString("   ")
		}
	}
	sb.WriteString("\n")

	// Body: weekday rows
	for day := 0; day < 7; day++ {
		sb.WriteString(fmt.Sprintf("[yellow]%-5s[-] ", weekdays[day]))
		for hour := 0; hour < 24; hour++ {
			val := matrix[day][hour]
			intensity := 0
			if maxValue > 0 && val > 0 {
				intensity = (val * (len(heatColors) - 1)) / maxValue
				if intensity >= len(heatColors) {
					intensity = len(heatColors) - 1
				}
			}
			sb.WriteString(fmt.Sprintf("[%s]██[-]", heatColors[intensity]))
		}
		sb.WriteString("\n")
	}

	// Legend
	sb.WriteString("\n      [gray]Low[-] ")
	for _, color := range heatColors {
		sb.WriteString(fmt.Sprintf("[%s]██[-]", color))
	}
	sb.WriteString(" [red]High[-]")

	return sb.String()
}

// RenderHeatmapCompact creates a more compact heatmap
func RenderHeatmapCompact(matrix [7][24]int, maxValue int) string {
	var sb strings.Builder

	// Header: hours (every 4 hours)
	sb.WriteString("     ")
	for h := 0; h < 24; h += 4 {
		sb.WriteString(fmt.Sprintf("[white]%02d[-]  ", h))
	}
	sb.WriteString("\n")

	// Body: weekday rows
	for day := 0; day < 7; day++ {
		sb.WriteString(fmt.Sprintf("[yellow]%-4s[-] ", weekdays[day][:3]))
		for hour := 0; hour < 24; hour++ {
			val := matrix[day][hour]
			intensity := 0
			if maxValue > 0 && val > 0 {
				intensity = (val * (len(heatColors) - 1)) / maxValue
				if intensity >= len(heatColors) {
					intensity = len(heatColors) - 1
				}
			}
			sb.WriteString(fmt.Sprintf("[%s]█[-]", heatColors[intensity]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetHeatmapStats returns summary statistics for the heatmap
func GetHeatmapStats(matrix [7][24]int) (peakDay int, peakHour int, totalCommits int) {
	maxVal := 0
	for day := 0; day < 7; day++ {
		for hour := 0; hour < 24; hour++ {
			totalCommits += matrix[day][hour]
			if matrix[day][hour] > maxVal {
				maxVal = matrix[day][hour]
				peakDay = day
				peakHour = hour
			}
		}
	}
	return
}
