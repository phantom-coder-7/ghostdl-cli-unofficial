package task

import (
	"fmt"
	"io"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func printMutationResult(out io.Writer, format, taskID, action, outcome, tableLine string) error {
	if format == "table" {
		fmt.Fprint(out, tableLine)
		return nil
	}
	return output.NewFormatter(out, format).Print(task.MutationResult{
		TaskID:  taskID,
		Action:  action,
		Outcome: outcome,
	})
}
