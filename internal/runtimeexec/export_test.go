package runtimeexec

import (
	"context"
	"io"
	"iter"
	"os"

	"github.com/metalagman/aida/internal/config"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/session"
)

var ContentTextForTest = contentText
var BuildUserPromptForTest = buildUserPrompt
var NormalizeCommandForTest = NormalizeCommand

func SetOSGetwdForTest(fn func() (string, error)) {
	osGetwd = fn
}

func ResetOSGetwdForTest() {
	osGetwd = os.Getwd
}

func SetRunAgentTurnFuncForTest(fn func(context.Context, adkagent.Agent, string) (string, error)) {
	runAgentTurnFunc = fn
}

func ResetRunAgentTurnFuncForTest() {
	runAgentTurnFunc = runAgentTurn
}

func SetBuildAgentRuntimeFuncForTest(
	fn func(context.Context, *config.Config, string, string, io.Writer) (adkagent.Agent, error),
) {
	buildAgentRuntimeFunc = fn
}

func ResetBuildAgentRuntimeFuncForTest() {
	buildAgentRuntimeFunc = buildAgentRuntime
}

func SeqFromEventsForTest(events ...*session.Event) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		for _, ev := range events {
			if !yield(ev, nil) {
				return
			}
		}
	}
}
