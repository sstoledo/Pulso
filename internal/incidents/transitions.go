package incidents

type TransitionAction struct {
	Open  bool
	Close bool
}

func Action(consecutiveFailures int, hasOpenIncident, success bool) TransitionAction {
	if success {
		return TransitionAction{Close: hasOpenIncident}
	}
	return TransitionAction{Open: consecutiveFailures >= 2 && !hasOpenIncident}
}
