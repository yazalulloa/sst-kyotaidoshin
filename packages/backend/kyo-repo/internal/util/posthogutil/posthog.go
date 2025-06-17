package posthogutil

type SeverityLevel string

const (
	SeverityFatal   SeverityLevel = "fatal"
	SeverityError   SeverityLevel = "error"
	SeverityWarning SeverityLevel = "warning"
	SeverityLog     SeverityLevel = "log"
	SeverityInfo    SeverityLevel = "info"
	SeverityDebug   SeverityLevel = "debug"
)

type ErrorProperties struct {
	ExceptionList             []Exception    `json:"exception_list"`
	ExceptionLevel            *SeverityLevel `json:"exception_level,omitempty"`
	ExceptionDOMExceptionCode *string        `json:"exception_DOMException_code,omitempty"`
	ExceptionPersonURL        *string        `json:"exception_personURL,omitempty"`
}

type Exception struct {
	Type       *string     `json:"type,omitempty"`
	Value      *string     `json:"value,omitempty"`
	Mechanism  *Mechanism  `json:"mechanism,omitempty"`
	Module     *string     `json:"module,omitempty"`
	ThreadId   *int64      `json:"thread_id,omitempty"`
	Stacktrace *Stacktrace `json:"stacktrace,omitempty"`
}

type Mechanism struct {
	Handled   *bool   `json:"handled,omitempty"`
	Type      *string `json:"type,omitempty"`
	Source    *string `json:"source,omitempty"`
	Synthetic *bool   `json:"synthetic,omitempty"`
}

type StackFrame struct {
	Platform        string          `json:"platform"`
	Filename        *string         `json:"filename,omitempty"`
	Function        *string         `json:"function,omitempty"`
	Module          *string         `json:"module,omitempty"`
	Lineno          *int            `json:"lineno,omitempty"`
	Colno           *int            `json:"colno,omitempty"`
	AbsPath         *string         `json:"abs_path,omitempty"`
	ContextLine     *string         `json:"context_line,omitempty"`
	PreContext      *[]string       `json:"pre_context,omitempty"`
	PostContext     *[]string       `json:"post_context,omitempty"`
	InApp           *bool           `json:"in_app,omitempty"`
	InstructionAddr *string         `json:"instruction_addr,omitempty"`
	AddrMode        *string         `json:"addr_mode,omitempty"`
	Vars            *map[string]any `json:"vars,omitempty"`
	ChunkId         *string         `json:"chunk_id,omitempty"`
}

type Stacktrace struct {
	Frames *[]StackFrame `json:"frames,omitempty"`
	Type   string        `json:"type"`
}
