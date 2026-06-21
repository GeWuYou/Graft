package demo

type WidgetDefinition struct {
	Title       string
	TitleKey    string
	Description string
}

func demoWidget() WidgetDefinition {
	return WidgetDefinition{
		Title:       "Dashboard title",
		Description: "Dashboard description",
	}
}
