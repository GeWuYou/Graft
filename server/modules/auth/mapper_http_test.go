package auth

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestToBootstrapResponsePreservesMenuSectionMetadata(t *testing.T) {
	response, err := toBootstrapResponse(moduleapi.AuthBootstrapPayload{
		User: moduleapi.CurrentUser{ID: 1},
		Menus: []moduleapi.AuthBootstrapMenuItem{{
			Code:            "docker",
			Kind:            "group",
			Title:           "Docker",
			SectionKey:      "runtime",
			SectionTitleKey: "menu.section.runtime",
			Icon:            "docker-provider",
		}},
	})
	if err != nil {
		t.Fatalf("map bootstrap response: %v", err)
	}
	if len(response.Menus) != 1 {
		t.Fatalf("expected one menu, got %#v", response.Menus)
	}
	menu := response.Menus[0]
	if menu.SectionKey == nil || *menu.SectionKey != "runtime" {
		t.Fatalf("expected runtime section key, got %#v", menu.SectionKey)
	}
	if menu.SectionTitleKey == nil || *menu.SectionTitleKey != "menu.section.runtime" {
		t.Fatalf("expected runtime section title key, got %#v", menu.SectionTitleKey)
	}
}
