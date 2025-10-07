package tasks

type SyncEngine interface {
	Run()
	Diff()
	Dump()
}

type PlaylistEngine struct{}

func (p *PlaylistEngine) Run()  {}
func (p *PlaylistEngine) Diff() {}
func (p *PlaylistEngine) Dump() {}
