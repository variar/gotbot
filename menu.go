package gotbot

type Menu struct {
	Name   string
	Items  []MenuEntry
	Parent *Menu
}

type MenuEntry struct {
	Name    string
	Command *CommandHandler
	Submenu *Menu
}

func NewMenu(name string) *Menu {
	return &Menu{Name: name, Items: make([]MenuEntry, 0), Parent: nil}
}

func (menu *Menu) AddCommand(name string, command *CommandHandler) *Menu {
	menu.Items = append(menu.Items, MenuEntry{Name: name, Command: command})
	return menu
}

func (menu *Menu) AddSumbenu(submenu *Menu) *Menu {
	menu.Items = append(menu.Items, MenuEntry{Name: submenu.Name, Submenu: submenu})
	submenu.Parent = menu
	return menu
}

func (menuEntry *MenuEntry) IsCommand() bool {
	return menuEntry.Command != nil
}
