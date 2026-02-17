package devconsole

type lineMsg struct {
	proc string
	line string
}

type lineBatchMsg struct {
	proc  string
	lines []string
}

type procExitMsg struct {
	proc string
	err  error
	code int
}

type procStartedMsg struct {
	proc string
	pid  int
}

type actionDoneMsg struct {
	message     string
	err         error
	markRunning []string
}
