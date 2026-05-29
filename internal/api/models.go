package api

type Tier struct {
	Name             string  `json:"name"`
	Tier             string  `json:"tier"`
	MsatsPerSecond   int64   `json:"msats_per_second"`
	SatsPerSecond    float64 `json:"sats_per_second"`
	USDCentsPerSec   float64 `json:"usd_cents_per_second"`
	CostPerMinSats   float64 `json:"cost_per_minute_sats"`
	CostPerHourSats  float64 `json:"cost_per_hour_sats"`
	CostPerDaySats   float64 `json:"cost_per_day_sats"`
	CostPerHourUSD   float64 `json:"cost_per_hour_usd"`
}

type SandboxStatus struct {
	SandboxID           string          `json:"sandbox_id"`
	ID                  string          `json:"ID"`
	Status              string          `json:"status"`
	Tier                string          `json:"tier"`
	Runtime             string          `json:"runtime"`
	BillingMethod       string          `json:"billing_method"`
	Template            string          `json:"template"`
	ConsumedMsats       int64           `json:"consumed_msats"`
	MsatsPerSecond      int64           `json:"msats_per_second"`
	SatsPerSecond       float64         `json:"sats_per_second"`
	PrepaidBalanceMsats int64           `json:"prepaid_balance_msats"`
	SecondsRemaining    int             `json:"seconds_remaining"`
	WarningLevel        string          `json:"warning_level"`
	NextPollSecs        int             `json:"next_poll_secs"`
	PendingInvoice      *PendingInvoice `json:"pending_invoice"`
	SandboxToken        string          `json:"sandbox_token"`
	Invoice             string          `json:"invoice"`
}

func (s *SandboxStatus) IDOrSandboxID() string {
	if s.SandboxID != "" {
		return s.SandboxID
	}
	return s.ID
}

type Execution struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	SandboxID   string `json:"sandbox_id"`
	Command     string `json:"command"`
}

type OutputChunk struct {
	Stream string `json:"stream"`
	Chunk  string `json:"chunk"`
}

type ExecResult struct {
	ExecutionID    string          `json:"execution_id"`
	Status         string          `json:"status"`
	Output         []OutputChunk   `json:"output"`
	ExitCode       int             `json:"exit_code"`
	Error          string          `json:"error"`
	PendingInvoice *PendingInvoice `json:"pending_invoice"`
}

type FileEntry struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	ModTS int64  `json:"mod_ts"`
	IsDir bool   `json:"is_dir"`
}

type PendingInvoice struct {
	Bolt11      string `json:"bolt11"`
	PaymentHash string `json:"payment_hash"`
	AmountMsats int64  `json:"amount_msats"`
	ExpiresAt   string `json:"expires_at"`
}

type CreateSandboxRequest struct {
	Tier              string `json:"tier"`
	BillingMethod     string `json:"billing_method,omitempty"`
	PrepaidSats       int64  `json:"prepaid_sats,omitempty"`
	InitialCreditsCents int  `json:"initial_credits_cents,omitempty"`
	Label             string `json:"label,omitempty"`
}

type ExecRequest struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
	TimeoutMs  int    `json:"timeout_ms,omitempty"`
}

type WriteFileRequest struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Encoding   string `json:"encoding,omitempty"`
	CreateDirs bool   `json:"create_dirs"`
}

type ReadFileRequest struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
	MaxBytes int    `json:"max_bytes,omitempty"`
}

type ReadFileResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
}
