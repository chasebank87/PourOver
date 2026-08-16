package discovery

// UnlinkStatusKind describes an explicit unlink path.
type UnlinkStatusKind string

const (
	UnlinkStatusMissing UnlinkStatusKind = "missing"
	UnlinkStatusRemove  UnlinkStatusKind = "remove"
)

// UnlinkStatus is the discovered state of one explicit unlink path.
type UnlinkStatus struct {
	Path       string // as declared in config
	TargetPath string // resolved absolute path
	Kind       UnlinkStatusKind
}
