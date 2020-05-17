package postman

import (
	"fmt"
	"path"
	"strings"

	"github.com/vroomy/service"
)

// Groups

func postmanGroupFrom(group *service.Group, groupMap map[string]*postmanGroup) (pg *postmanGroup, created bool) {
	if oldGroup, ok := groupMap[group.Name]; ok {
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

func addGroupToParent(g *postmanGroup, parent string, groupMap map[string]*postmanGroup) (err error) {
	if parentGroup, ok := groupMap[parent]; ok {
		// Add group to parent item list
		g.route = path.Join(parentGroup.route, g.route)
		parentGroup.Item = append(parentGroup.Item, g)
	} else {
		err = fmt.Errorf("Unable to find parent group: %s", parentGroup)
		return
	}

	return
}

// Routes

func postmanRouteFrom(route *service.Route) (r *postmanRoute) {
	r = &postmanRoute{}
	r.Name = r.route

	r.Request = &postmanRequest{
		Method: route.Method,
		URL: &postmanURL{
			Raw:  "{{URL}}" + r.route,
			Host: "{{URL}}",
			Path: strings.Split(strings.Trim(r.route, "/"), "/"),
		},
	}

	r.route = route.HTTPPath
	return
}

func addRouteToParent(route *service.Route, groupMap map[string]*postmanGroup) {
	if parentGroup, ok := groupMap[route.Group]; ok {
		var r *postmanRoute
		r = postmanRouteFrom(route)

		// Prepend parent route for full path
		r.route = path.Join(parentGroup.route, r.route)

		// Add route to group
		parentGroup.Item = append(parentGroup.Item, r)
	} else {
		out.Warningf("Unable to find parent (%s) of route: %s", route.Group, route.HTTPPath)
	}
}
