package plugins

type pluginslice []*Plugin

func (ps pluginslice) getIndex(alias string) (index int) {
	for i, p := range ps {
		if p.alias == alias {
			return i
		}
	}

	return -1
}

func (ps pluginslice) get(alias string) (p *Plugin, ok bool) {
	idx := ps.getIndex(alias)
	if idx == -1 {
		return
	}

	p = ps[idx]
	ok = true
	return
}

func (ps pluginslice) append(p *Plugin) (out pluginslice, err error) {
	idx := ps.getIndex(p.alias)
	if idx > -1 {
		err = ErrPluginKeyExists
		return
	}

	out = append(ps, p)
	return
}
