package gotbot

type ReplySender interface {
	SendReply(reply string)
	AskOptions(reply string, options []string, force bool)
	FirstName() string
}

type Location struct {
	Lon float64
	Lat float64
}

type TextParameterParser func(input string) (string, error)
type LocationParameterParser func(loc Location) (string, error)

type CommandParameter struct {
	Name          string
	AskQuestion   string
	ParseText     TextParameterParser
	ParseLocation LocationParameterParser
}

type ProcessCommand func(parsedParams map[string]string, replySender ReplySender)

type CommandHandler struct {
	Name    string
	params  []*CommandParameter
	process ProcessCommand
}

func NewCommandHandler(name string, processer ProcessCommand) *CommandHandler {
	handler := CommandHandler{name, make([]*CommandParameter, 0), processer}
	return &handler
}

func (command *CommandHandler) AddParameter(parameter *CommandParameter) *CommandHandler {
	command.params = append(command.params, parameter)
	return command
}
