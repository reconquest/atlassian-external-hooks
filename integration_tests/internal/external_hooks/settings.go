package external_hooks

type Settings interface {
	Safe() bool
	Exe() string
	Params() []string
}

type ScopeSettings struct {
	ValueSafe   bool
	ValueExe    string
	ValueParams []string
}

var (
	_ Settings = (*ScopeSettings)(nil)
	_ Settings = (*GlobalSettings)(nil)
)

func IsGlobalSettings(settings Settings) bool {
	_, ok := settings.(*GlobalSettings)
	return ok
}

func NewScopeSettings() *ScopeSettings {
	return &ScopeSettings{}
}

func (settings *ScopeSettings) UseSafePath(enabled bool) *ScopeSettings {
	settings.ValueSafe = enabled

	return settings
}

func (settings *ScopeSettings) WithExe(exe string) *ScopeSettings {
	settings.ValueExe = exe

	return settings
}

func (settings *ScopeSettings) WithParams(args ...string) *ScopeSettings {
	settings.ValueParams = args

	return settings
}

func (settings *ScopeSettings) Exe() string {
	return settings.ValueExe
}

func (settings *ScopeSettings) Safe() bool {
	return settings.ValueSafe
}

func (settings *ScopeSettings) Params() []string {
	return settings.ValueParams
}

type FilterPersonalRepositories int

const (
	FILTER_PERSONAL_REPOSITORIES_DISABLED         FilterPersonalRepositories = 0
	FILTER_PERSONAL_REPOSITORIES_ONLY_PERSONAL    FilterPersonalRepositories = 1
	FILTER_PERSONAL_REPOSITORIES_EXCLUDE_PERSONAL FilterPersonalRepositories = 2
)

type GlobalSettings struct {
	ValueSafe                       bool
	ValueExe                        string
	ValueParams                     []string
	ValueFilterPersonalRepositories FilterPersonalRepositories
}

func NewGlobalSettings() *GlobalSettings {
	return &GlobalSettings{}
}

func (settings *GlobalSettings) WithFilterPersonalRepositories(
	value FilterPersonalRepositories,
) *GlobalSettings {
	settings.ValueFilterPersonalRepositories = value
	return settings
}

func (settings *GlobalSettings) Exe() string {
	return settings.ValueExe
}

func (settings *GlobalSettings) Safe() bool {
	return settings.ValueSafe
}

func (settings *GlobalSettings) Params() []string {
	return settings.ValueParams
}

func (settings *GlobalSettings) FilterPersonalRepositories() FilterPersonalRepositories {
	return settings.ValueFilterPersonalRepositories
}

func (settings *GlobalSettings) UseSafePath(enabled bool) *GlobalSettings {
	settings.ValueSafe = enabled

	return settings
}

func (settings *GlobalSettings) WithExe(exe string) *GlobalSettings {
	settings.ValueExe = exe

	return settings
}

func (settings *GlobalSettings) WithParams(args ...string) *GlobalSettings {
	settings.ValueParams = args

	return settings
}
