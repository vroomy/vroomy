package postman

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/hatchify/scribe"
	"github.com/vroomy/service"
)

// PostmanSchema defines the current struct definitions
const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
const rootRoute = "/"

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

	var rootGroup = &postmanGroup{Name: "/", route: rootRoute}

	p = &Postman{
		Info: postmanInfo{
			Name:   cfg.Name + " (vroomy)",
			Schema: postmanSchema,
		},
		Item:     postmanCollection{rootGroup},
		groupMap: map[string]*postmanGroup{rootGroup.route: rootGroup},
	}

	// Load nested groups
	var groupCount = 0
	for _, group := range cfg.Groups {
		p.addGroup(group)
		groupCount++
	}

	// Load route and add to parent group
	var routeCount = 0
	for _, route := range cfg.Routes {
		p.addRoute(route)
		routeCount++
	}

	// Check if root group is necessary
	var rootItemCount = len(rootGroup.Item)
	if rootItemCount == 0 {
		// No root routes, remove root group
		p.Item = p.Item[1:]
	} else {
		out.Notificationf("%d ungrouped routes found in root \"/\" path.", rootItemCount)
	}

	out.Successf("Documented %d routes in %d groups for %s!", routeCount, groupCount, p.Info.Name)
	return
}

// WriteToFile will output the postman config json to the provided filename
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

// Groups

func (p *Postman) addGroup(group *service.Group) {
	group.Name = strings.Trim(group.Name, "/")
	group.Group = strings.Trim(group.Group, "/")

	var g *postmanGroup
	var newGroup bool
	g, newGroup = postmanGroupFrom(group, p.groupMap)

	// Check if group has parent
	g.route = group.HTTPPath
	if len(group.Group) == 0 {
		if newGroup {
			// Root group
			p.Item = append(p.Item, g)
			out.Notificationf("Found root group \"%s\"", g.Name)
		} else {
			// We've seen this group before... Is this a problem?
			out.Warningf("Repeat definition of group \"%s\"", g.Name)
		}
	} else {
		// Group is nested in a parent group
		if err := addGroupToParent(g, group.Group, p.groupMap); err != nil {
			out.Error(err.Error())
		} else {
			out.Notificationf("Added group \"%s\" to parent \"%s\"", g.Name, group.Group)
		}
	}

	// Add group to quick-reference map
	p.groupMap[group.Name] = g
}

// Routes

func (p *Postman) addRoute(route *service.Route) {
	route.HTTPPath = strings.Trim(route.HTTPPath, "/")
	route.Group = strings.Trim(route.Group, "/")

	// Add to root group
	if len(route.Group) == 0 {
		route.Group = rootRoute
	}

	if len(route.Group) > 0 {
		addRouteToParent(route, p.groupMap)
	} else {
		// Orphan route!
		out.Errorf("Unable to find parent (%s) of route: %s", route.Group, route.HTTPPath)
	}
}
