package overview

import "github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"

// Response is the overview snapshot returned by ghostdl overview.
type Response struct {
	Auth                    AuthBlock    `json:"auth"`
	Counts                  StatusCounts `json:"counts"`
	DownloadRateBytesPerSec int64        `json:"downloadRateBytesPerSec"`
	ReceivedBytesRunning    int64        `json:"receivedBytesRunning"`
	BtRunningCount          int          `json:"btRunningCount"`
	InProgress              []*task.Task `json:"inProgress"`
	Failed                  []*task.Task `json:"failed"`
	RecentCompleted         []*task.Task `json:"recentCompleted"`
}

// AuthBlock describes server and client versions for the overview header.
type AuthBlock struct {
	ServerURL  string `json:"serverUrl"`
	AppVersion string `json:"appVersion"`
	CLIVersion string `json:"cliVersion"`
}

// StatusCounts holds task counts by status from the full snapshot.
type StatusCounts struct {
	Waiting   int `json:"waiting"`
	Running   int `json:"running"`
	Paused    int `json:"paused"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}
