package task

// Task represents a Ghost Downloader download task.
type Task struct {
	ID            string  `json:"id"            table:"ID"              view:"overview"`
	Name          string  `json:"name"          table:"Name"            view:"overview"`
	Status        Status  `json:"status"        table:"Status"          view:"overview"`
	Progress      float64 `json:"progress"      table:"Progress%"       view:"overview"`
	ReceivedBytes int64   `json:"receivedBytes" table:"Received"`
	FileSize      int64   `json:"fileSize"      table:"Size"            view:"overview"`
	Speed         int64   `json:"speed"         table:"Speed"`
	CreatedAt     string  `json:"createdAt"     table:"Created At"`
	CanPause      bool    `json:"canPause"      table:"Can Pause"`
	CanOpenFile   bool    `json:"canOpenFile"   table:"Can Open File"`
	CanOpenFolder bool    `json:"canOpenFolder" table:"Can Open Folder"`
	FileExt       string  `json:"fileExt"       table:"File Ext"`
	PackName      string  `json:"packName"      table:"Pack Name"`
	URL           string  `json:"url,omitempty" table:"URL"`
}

type Status string

const (
	StatusWaiting   Status = "waiting"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type CreateRequest struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename,omitempty"`
	Path     string            `json:"path,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Threads  int               `json:"threads,omitempty"`
	Title    string            `json:"title,omitempty"`
	Source   string            `json:"source"`
	Draft    bool              `json:"draft,omitempty"`
}

type CreateResult struct {
	Status  string `json:"status"`
	TaskID  string `json:"taskId,omitempty"`
	Message string `json:"message,omitempty"`
}

type ListRequest struct {
	Status   Status   `json:"status,omitempty"`
	Statuses []Status `json:"statuses,omitempty"`
	Packs    []string `json:"packs,omitempty"`
	Exts     []string `json:"exts,omitempty"`
	Query    string   `json:"query,omitempty"`
	Sort     string   `json:"sort,omitempty"`
	Order    string   `json:"order,omitempty"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
}

type ListResponse struct {
	Tasks []*Task `json:"tasks"`
	Total int     `json:"total"`
}

type ProgressResponse struct {
	TaskID   string  `json:"task_id"     table:"Task ID"`
	Total    int64   `json:"total"       table:"Total"`
	Consumed int64   `json:"consumed"    table:"Consumed"`
	Progress float64 `json:"progress"    table:"Progress"`
	Status   Status  `json:"status"      table:"Status"`
	ETA      int64   `json:"eta_seconds" table:"ETA (seconds)"`
}

type MutationResult struct {
	TaskID  string `json:"taskId" table:"Task ID"`
	Action  string `json:"action" table:"Action"`
	Outcome string `json:"outcome" table:"Outcome"`
}
