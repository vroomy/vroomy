package postman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/hatchify/scribe"
	"github.com/vroomy/service"
)

// PostmanSchema defines the current struct definitions
const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// Postman represents the documenation output object
type Postman struct {
	Info postmanInfo       `json:"info,omitempty"`
	Item postmanCollection `json:"item,omitempty"`

	groupMap map[string]*postmanGroup
}

// PostmanCollection represents the generated api groups and their route trees
type postmanCollection []*postmanGroup

// Postman info defines the schema and collection meta
type postmanInfo struct {
	Name      string `json:"name,omitempty"`
	Schema    string `json:"schema,omitempty"`
	PostmanID string `json:"_postman_id,omitempty"`
}

// PostmanGroup translates from service.Group
type postmanGroup struct {
	Name string        `json:"name,omitempty"`
	Item []interface{} `json:"item,omitempty"`

	route string
}

// PostmanRoute translates from service.Route
type postmanRoute struct {
	Name     string            `json:"name,omitempty"`
	Request  *postmanRequest   `json:"request,omitempty"`
	Response []postmanResponse `json:"response,omitempty"`

	route string
}

// PostmanRequest is generated from vroomy config group/route tree
type postmanRequest struct {
	Method   string           `json:"method,omitempty"`
	Header   []postmanHeader  `json:"header,omitempty"`
	Body     *postmanBody     `json:"body,omitempty"`
	URL      *postmanURL      `json:"url,omitempty"`
	Response *postmanResponse `json:"response,omitempty"`
}

// PostmanURL represents the different url components supported in schema
type postmanURL struct {
	Raw   string      `json:"raw,omitempty"`
	Host  string      `json:"host,omitempty"`
	Path  []string    `json:"path,omitempty"`
	Query []postmanKV `json:"query,omitempty"`
}

// PostmanResponse is an example response for a given request
type postmanResponse struct {
}

// PostmanKV is standard kv representation
type postmanKV struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// PostmanBody represents the different styles of POST body supported
type postmanBody struct {
	Mode    string         `json:"mode,omitempty"`
	Raw     string         `json:"raw,omitempty"`
	Options postmanOptions `json:"options,omitempty"`
}

// PostmanHeader is a named/typed KV
type postmanHeader struct {
	postmanKV

	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type postmanOptions map[string]map[string]string

var out *scribe.Scribe

// FromConfig returns Postman object parsed from vroomy config
func FromConfig(cfg *service.Config) (p *Postman, err error) {
	out = scribe.New("Postman Docs")
	out.Notification("Parsing vroomy config...")

	var rootGroup = &postmanGroup{Name: "/", route: "/"}

	p = &Postman{
		Info: postmanInfo{
			Name:   cfg.Name + " vroomy",
			Schema: postmanSchema,
		},
		Item:     postmanCollection{rootGroup},
		groupMap: map[string]*postmanGroup{rootGroup.route: rootGroup},
	}

	for _, group := range cfg.Groups {
		p.addGroup(group)
	}

	for _, route := range cfg.Routes {
		p.addRoute(route)
	}

	if len(rootGroup.Item) == 0 {
		// No root routes, remove root group
		p.Item = p.Item[1:]
	}

	return
}

func (p *Postman) WriteToFile(filename string) (err error) {
	var bytes []byte
	if bytes, err = json.MarshalIndent(p, "", "  "); err != nil {
		return
	}

	if err = ioutil.WriteFile(filename, bytes, 0644); err != nil {
		out.Error(err.Error())
		return
	}

	return
}

func (p *Postman) groupFrom(group *service.Group) (pg *postmanGroup, created bool) {
	if oldGroup, ok := p.groupMap[group.Name]; ok {
		// Append to existing group
		pg = oldGroup
		created = false
	} else {
		// New group
		pg = &postmanGroup{Name: group.Name}
		created = true
	}

	return
}

func (p *Postman) addGroupToParent(g *postmanGroup, parent string) (err error) {
	if parent, ok := p.groupMap[parent]; ok {
		g.route = path.Join(parent.route, g.route)
		parent.Item = append(parent.Item, g)
	} else {
		err = fmt.Errorf("Unable to find parent group: %s", parent)
		return
	}

	return
}

func (p *Postman) addGroup(group *service.Group) {
	group.Name = strings.Trim(group.Name, "/")
	group.Group = strings.Trim(group.Group, "/")

	var g *postmanGroup
	var newGroup bool
	g, newGroup = p.groupFrom(group)

	g.route = group.HTTPPath
	if len(group.Group) == 0 {
		if newGroup {
			// Root group
			p.Item = append(p.Item, g)
			out.Notificationf("Found root group \"%s\"", g.Name)
		} else {
			out.Warningf("Repeat definition of group \"%s\"", g.Name)
		}
	} else {
		if err := p.addGroupToParent(g, group.Group); err != nil {
			out.Warning(err.Error())
		} else {
			out.Notificationf("Added group \"%s\" to parent \"%s\"", g.Name, group.Group)
		}
	}

	p.groupMap[group.Name] = g
}

func (p *Postman) addRouteToParent(route *service.Route) {
	var r *postmanRoute
	r = &postmanRoute{}
	r.route = route.HTTPPath

	if parent, ok := p.groupMap[route.Group]; ok {
		r.route = path.Join(parent.route, r.route)
		r.Name = r.route
		r.Request = &postmanRequest{
			Method: route.Method,
			URL: &postmanURL{
				Raw:  "{{URL}}" + r.route,
				Host: "{{URL}}",
				Path: strings.Split(strings.Trim(r.route, "/"), "/"),
			},
		}

		parent.Item = append(parent.Item, r)
	} else {
		out.Warningf("Unable to find parent (%s) of route: %s", route.Group, route.HTTPPath)
	}
}

func (p *Postman) addRoute(route *service.Route) {
	route.HTTPPath = strings.Trim(route.HTTPPath, "/")
	route.Group = strings.Trim(route.Group, "/")

	if len(route.Group) == 0 {
		route.Group = "/"
	}

	if len(route.Group) > 0 {
		p.addRouteToParent(route)
	} else {
		out.Warningf("Unable to find parent (%s) of route: %s", route.Group, route.HTTPPath)
	}
}
