package main

import (
	"path"
	"strings"

	"github.com/vroomy/vroomy/service"
)

type postmanGroup struct {
	Name string        `json:"name,omitempty"`
	Item []interface{} `json:"item,omitempty"`

	route string
}

type postmanRoute struct {
	Name     string            `json:"name,omitempty"`
	Request  *postmanRequest   `json:"request,omitempty"`
	Response []postmanResponse `json:"response,omitempty"`

	route string
}

type postmanRequest struct {
	Method   string           `json:"method,omitempty"`
	Header   []postmanHeader  `json:"header,omitempty"`
	Body     *postmanBody     `json:"body,omitempty"`
	URL      *postmanURL      `json:"url,omitempty"`
	Response *postmanResponse `json:"response,omitempty"`
}

type postmanURL struct {
	Raw   string      `json:"raw,omitempty"`
	Host  string      `json:"host,omitempty"`
	Path  []string    `json:"path,omitempty"`
	Query []postmanKV `json:"query,omitempty"`
}

type postmanResponse struct {
}

type postmanKV struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type postmanBody struct {
	Mode    string         `json:"mode,omitempty"`
	Raw     string         `json:"raw,omitempty"`
	Options postmanOptions `json:"options,omitempty"`
}

type postmanHeader struct {
	Key   string `json:"key,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type postmanOptions map[string]map[string]string

type postmanCollection []*postmanGroup

var groupCache = map[string]*postmanGroup{}

func postmanFromConfig() (output map[string]interface{}) {
	var (
		rootGroup  = &postmanGroup{Name: "/", route: "/"}
		collection = postmanCollection{}
	)

	groupCache[rootGroup.Name] = rootGroup

	for _, group := range cfg.Groups {
		collection.addGroup(group)
	}

	for _, route := range cfg.Routes {
		collection.addRoute(route)
	}

	output = map[string]interface{}{
		"info": map[string]string{
			"name":   cfg.Name + " vroomy",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},

		"item": collection,
	}

	return output
}

func (c *postmanCollection) addGroup(group *service.Group) {
	group.Name = strings.Trim(group.Name, "/")
	group.Group = strings.Trim(group.Group, "/")

	var g *postmanGroup
	var newGroup = false
	if oldGroup, ok := groupCache[group.Name]; ok {
		// Append to existing group
		g = oldGroup
		newGroup = false
	} else {
		// New group
		g = &postmanGroup{Name: group.Name}
		newGroup = true
	}

	g.route = group.HTTPPath
	if len(group.Group) > 0 {
		if parent, ok := groupCache[group.Group]; ok {
			g.route = path.Join(parent.route, g.route)
			parent.Item = append(parent.Item, g)
		} else {
			out.Warningf("Unable to find parent group: %s", group.Group)
			return
		}
	} else {
		// Root group
		if newGroup {
			*c = append(*c, g)
		}
	}

	groupCache[group.Name] = g
}

func (c *postmanCollection) addRoute(route *service.Route) {
	route.HTTPPath = strings.Trim(route.HTTPPath, "/")
	route.Group = strings.Trim(route.Group, "/")

	if len(route.Group) == 0 {
		route.Group = "/"
	}

	var r *postmanRoute
	r = &postmanRoute{}
	r.route = route.HTTPPath
	if len(route.Group) > 0 {
		if parent, ok := groupCache[route.Group]; ok {
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
	} else {
		out.Warningf("Unable to find parent (%s) of route: %s", route.Group, route.HTTPPath)
	}
}
