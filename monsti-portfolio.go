// This file is part of Monsti-Portfolio.
// Copyright 2015 Christian Neumann
//
// Monsti-Portfolio is free software: you can redistribute it and/or
// modify it under the terms of the GNU Affero General Public License
// as published by the Free Software Foundation, either version 3 of
// the License, or (at your option) any later version.
//
// Monsti-Portfolio is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti-Portfolio.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"

	"sort"
	"pkg.monsti.org/monsti/api/service"
	"pkg.monsti.org/monsti/api/util/i18n"
	"pkg.monsti.org/monsti/api/util/module"
	"pkg.monsti.org/monsti/api/util/nodes"
	"pkg.monsti.org/monsti/api/util/settings"
	mtemplate "pkg.monsti.org/monsti/api/util/template"
)

var availableLocales = []string{"de", "en"}

type portfolioEntryCtx struct {
	*service.Node
	Image *service.Node
}

func getPortfolio(req *service.Request, s *service.Session) (
	[]portfolioEntryCtx, error) {
	m := s.Monsti()
	portfolio, err := m.GetChildren(req.Site, req.NodePath)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch children: %v", err)
	}
	order := func(left, right *service.Node) bool {
		return left.Order < right.Order
	}
	sort.Sort(sort.Reverse(&nodes.Sorter{portfolio, order}))

	portfolioEntryCtxs := make([]portfolioEntryCtx, len(portfolio))
	for idx := range portfolio {
		portfolioEntryCtxs[idx].Node = portfolio[idx]
	}
	return portfolioEntryCtxs, nil
}

/*
func getPortfolioEntryContext(reqId uint, embed *service.EmbedNode,
	s *service.Session, m *settings.Monsti, renderer *mtemplate.Renderer) (
	map[string][]byte, *service.CacheMods, error) {
  m := s.Monsti()
	req, err := m.GetRequest(reqId)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not get request: %v", err)
	}
	rendered, err := renderer.Render("portfolio/Entry-images",
		mtemplate.Context{"Images": images},
		req.Session.Locale, m.GetSiteTemplatesPath(req.Site))
	if err != nil {
		return nil, nil, fmt.Errorf("Could not render template: %v", err)
	}
	mods := &service.CacheMods{
		Deps: []service.CacheDep{{Node: req.NodePath, Descend: 1}},
	}
	return map[string][]byte{"PortfolioEntryImages": rendered}, mods, nil
}
*/

func getPortfolioContext(reqId uint, embed *service.EmbedNode,
	s *service.Session, m *settings.Monsti, renderer *mtemplate.Renderer) (
	map[string][]byte, *service.CacheMods, error) {
	req, err := s.Monsti().GetRequest(reqId)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not get request: %v", err)
	}
	context := mtemplate.Context{}
	context["Entries"], err = getPortfolio(req, s)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not retrieve portfolio: %v", err)
	}
	context["Embedded"] = embed

	rendered, err := renderer.Render("portfolio/list", context,
		req.Session.Locale, m.GetSiteTemplatesPath(req.Site))
	if err != nil {
		return nil, nil, fmt.Errorf("Could not render template: %v", err)
	}
	mods := &service.CacheMods{
		Deps: []service.CacheDep{{Node: req.NodePath, Descend: 1}},
	}
	return map[string][]byte{"PortfolioList": rendered}, mods, nil
}

func setup(c *module.ModuleContext) error {
	G := func(in string) string { return in }
	m := c.Session.Monsti()

	nodeType := service.NodeType{
		Id:        "portfolio.Entry",
		AddableTo: []string{"portfolio.List"},
		Name:      i18n.GenLanguageMap(G("Portfolio Entry"), availableLocales),
		Hide:      true,
		Fields: []*service.FieldConfig{
			{Id: "core.Title"},
			{Id: "core.Description"},
			{Id: "core.Thumbnail"},
			{Id: "core.Body"},
		},
	}
	if err := m.RegisterNodeType(&nodeType); err != nil {
		return fmt.Errorf("Could not register %q node type: %v", nodeType.Id, err)
	}

	nodeType = service.NodeType{
		Id:        "portfolio.List",
		AddableTo: nil,
		Name:      i18n.GenLanguageMap(G("Portfolio List"), availableLocales),
		Fields: []*service.FieldConfig{
			{Id: "core.Title"},
			{Id: "core.Description"},
			{Id: "core.Thumbnail"},
			{Id: "core.Body"},
		},
	}
	if err := m.RegisterNodeType(&nodeType); err != nil {
		return fmt.Errorf("Could not register %q node type: %v", nodeType.Id, err)
	}

	handler := service.NewNodeContextHandler(c.Sessions,
		func(req uint, session *service.Session, nodeType string,
			embedNode *service.EmbedNode) (
			map[string][]byte, *service.CacheMods, error) {
			session, err := c.Sessions.New()
			if err != nil {
				return nil, nil, fmt.Errorf("Could not get session: %v", err)
			}
			defer c.Sessions.Free(session)
			switch nodeType {
			case "portfolio.List":
				ctx, mods, err := getPortfolioContext(req, embedNode, session, c.Settings,
					c.Renderer)
				if err != nil {
					return nil, nil, fmt.Errorf("Could not get portfolio context: %v", err)
				}
				return ctx, mods, nil
				/*
					case "portfolio.PortfolioEntry":
						ctx, mods, err := getPortfolioEntryContext(req, embedNode, session, c.Settings,
							c.Renderer)
						if err != nil {
							return nil, nil, fmt.Errorf("Could not get portfolioEntry context: %v", err)
						}
						return ctx, mods, nil
				*/
			default:
				return nil, nil, nil
			}
		})
	if err := m.AddSignalHandler(handler); err != nil {
		c.Logger.Fatalf("Could not add signal handler: %v", err)
	}
	return nil
}

func main() {
	module.StartModule("portfolio", setup)
}
