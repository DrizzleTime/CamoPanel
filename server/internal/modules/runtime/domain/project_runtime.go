package domain

const (
	TemplateIDOpenResty = "openresty"

	ActionStart    = "start"
	ActionStop     = "stop"
	ActionRestart  = "restart"
	ActionDelete   = "delete"
	ActionRedeploy = "redeploy"
)

type ManagedProject struct {
	ID          string
	Name        string
	TemplateID  string
	ComposePath string
	Status      string
	LastError   string
}
