package main

import (
	"path"
	"strings"
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

type postmanOptions map[string]map[string]string

type postmanHeader struct {
	Key   string `json:"key,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

func postmanFromConfig() (output map[string]interface{}) {
	var groups = map[string]*postmanGroup{}
	var postmanGroups = []*postmanGroup{}
	var g *postmanGroup
	var newGroup = false
	for _, group := range cfg.Groups {
		group.Name = strings.Trim(group.Name, "/")
		group.Group = strings.Trim(group.Group, "/")

		if oldGroup, ok := groups[group.Name]; ok {
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
			if parent, ok := groups[group.Group]; ok {
				g.route = path.Join(parent.route, g.route)
				parent.Item = append(parent.Item, g)
			} else {
				out.Warningf("Unable to find parent group: %s", group.Group)
				continue
			}
		} else {
			// Root group
			if newGroup {
				postmanGroups = append(postmanGroups, g)
			}
		}

		groups[group.Name] = g
	}

	var r *postmanRoute
	for _, route := range cfg.Routes {
		route.HTTPPath = strings.Trim(route.HTTPPath, "/")
		route.Group = strings.Trim(route.Group, "/")

		r = &postmanRoute{Name: route.HTTPPath}
		r.route = route.HTTPPath
		if len(route.Group) > 0 {
			if parent, ok := groups[route.Group]; ok {
				r.route = path.Join(parent.route, r.route)
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
				continue
			}
		}
	}

	output["info"] = map[string]string{"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"}
	output["item"] = postmanGroups

	return output
}
