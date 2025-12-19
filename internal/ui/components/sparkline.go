package components

import (
	"strings"
)

// Sparkline characters: U+2581 to U+2588
var sparkBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline converts values to unicode sparkline
func RenderSparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var sb strings.Builder
	scale := float64(len(sparkBars) - 1)
	if max > min {
		scale /= float64(max - min)
	} else {
		// All values are the same, use middle bar
		for range values {
			sb.WriteRune(sparkBars[len(sparkBars)/2])
		}
		return sb.String()
	}

	for _, v := range values {
		idx := int(float64(v-min) * scale)
		if idx >= len(sparkBars) {
			idx = len(sparkBars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		sb.WriteRune(sparkBars[idx])
	}

	return sb.String()
}

// RenderSparklineColored returns sparkline with tview color tags
func RenderSparklineColored(values []int, color string) string {
	spark := RenderSparkline(values)
	return "[" + color + "]" + spark + "[-]"
}

// RenderSparklineWithWidth renders sparkline scaled to a target width
func RenderSparklineWithWidth(values []int, targetWidth int) string {
	if len(values) == 0 || targetWidth <= 0 {
		return ""
	}

	if len(values) <= targetWidth {
		return RenderSparkline(values)
	}

	// Downsample by averaging
	scaledValues := make([]int, targetWidth)
	bucketSize := float64(len(values)) / float64(targetWidth)

	for i := 0; i < targetWidth; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(values) {
			end = len(values)
		}

		sum := 0
		for j := start; j < end; j++ {
			sum += values[j]
		}
		if end > start {
			scaledValues[i] = sum / (end - start)
		}
	}

	return RenderSparkline(scaledValues)
}
