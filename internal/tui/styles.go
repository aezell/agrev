package tui

import "github.com/charmbracelet/lipgloss"

// Color palette.
var (
	colorRed       = lipgloss.Color("#ff5555")
	colorGreen     = lipgloss.Color("#50fa7b")
	colorYellow    = lipgloss.Color("#f1fa8c")
	colorBlue      = lipgloss.Color("#8be9fd")
	colorPurple    = lipgloss.Color("#bd93f9")
	colorDim       = lipgloss.Color("#6272a4")
	colorBg        = lipgloss.Color("#282a36")
	colorBgLight   = lipgloss.Color("#343746")
	colorFg        = lipgloss.Color("#f8f8f2")
	colorOrange    = lipgloss.Color("#ffb86c")
	colorBorder    = lipgloss.Color("#44475a")
	colorHighlight = lipgloss.Color("#44475a")
)

// Style definitions.
var (
	// File list styles
	fileListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	fileItemStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	fileItemSelectedStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorHighlight).
				Bold(true)

	fileItemNewStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	fileItemDeletedStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	// Diff view styles
	diffViewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Width(4).
			Align(lipgloss.Right)

	addedLineStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	deletedLineStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	contextLineStyle = lipgloss.NewStyle().
				Foreground(colorFg)

	hunkHeaderStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	fileHeaderStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true).
			Padding(0, 0, 1, 0)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorBgLight).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Background(colorBgLight).
			Bold(true)

	// Trace panel styles
	traceViewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	traceHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true).
				Padding(0, 0, 1, 0)

	traceWriteStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	traceBashStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	traceReasonStyle = lipgloss.NewStyle().
				Foreground(colorFg)

	traceReadStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	traceUserStyle = lipgloss.NewStyle().
			Foreground(colorPurple)

	// Finding annotation styles
	findingHighStyle = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true)

	findingMediumStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	findingLowStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	// Review decision styles
	fileApprovedStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	fileRejectedStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Bold(true)

	filePendingStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	summaryHeaderStyle = lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true).
				Padding(1, 0)

	summaryApprovedStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	summaryRejectedStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	summaryPendingStyle = lipgloss.NewStyle().
				Foreground(colorYellow)

	// Help bar
	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorYellow)
)
